package sqlstore

import (
	"encoding/json"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	kvMigrationDoneKey = "KVMigrationDone"
	kvMigrationDoneVal = "true"

	kvStoreListPrefix  = "order_"
	kvStoreIssuePrefix = "item_"
	kvListPageSize     = 100
)

// kvIssue mirrors the legacy Issue struct stored in KV.
type kvIssue struct {
	ID            string `json:"id"`
	Message       string `json:"message"`
	PostPermalink string `json:"postPermalink"`
	Description   string `json:"description,omitempty"`
	CreateAt      int64  `json:"create_at"`
	PostID        string `json:"post_id"`
}

// kvIssueRef mirrors the legacy IssueRef struct stored in KV list arrays.
type kvIssueRef struct {
	IssueID        string `json:"issue_id"`
	ForeignIssueID string `json:"foreign_issue_id"`
	ForeignUserID  string `json:"foreign_user_id"`
}

// MigrateFromKV reads all KV entries for the Todo plugin and inserts them
// into the todo_items table. It is idempotent: if a migration was already
// completed (flag in TODO_System), it returns immediately.
//
// The migration strategy:
//  1. Scan KV keys via KVList.
//  2. For each "order_{userID}{listSuffix}" key, decode the []*kvIssueRef.
//  3. For each ref, load "item_{issueID}" to get the issue body.
//  4. INSERT the combined row into todo_items.
//  5. Record migration-done flag in TODO_System.
//
// Existing KV data is NOT deleted (safe rollback).
func (s *SQLStore) MigrateFromKV(api plugin.API) error {
	done, err := s.getSystemValue(s.db, kvMigrationDoneKey)
	if err != nil {
		logrus.WithError(err).Warn("Could not check KV migration flag, proceeding with migration")
	}
	if done == kvMigrationDoneVal {
		logrus.Info("KV->DB migration already completed, skipping")
		return nil
	}

	logrus.Info("Starting KV->DB migration for Todo plugin")

	keys, err := getAllKVKeys(api)
	if err != nil {
		return errors.Wrap(err, "failed to list KV keys")
	}

	issueCache := make(map[string]*kvIssue)
	listEntries := make([]listEntry, 0)

	for _, key := range keys {
		switch {
		case strings.HasPrefix(key, kvStoreIssuePrefix):
			issueID := strings.TrimPrefix(key, kvStoreIssuePrefix)
			issue, loadErr := loadKVIssue(api, key)
			if loadErr != nil {
				logrus.WithError(loadErr).WithField("key", key).Warn("Skipping unreadable issue")
				continue
			}
			issueCache[issueID] = issue

		case strings.HasPrefix(key, kvStoreListPrefix):
			entry, parseErr := parseListKey(key)
			if parseErr != nil {
				logrus.WithError(parseErr).WithField("key", key).Warn("Skipping unparseable list key")
				continue
			}

			refs, loadErr := loadKVList(api, key)
			if loadErr != nil {
				logrus.WithError(loadErr).WithField("key", key).Warn("Skipping unreadable list")
				continue
			}
			entry.refs = refs
			listEntries = append(listEntries, entry)
		}
	}

	tx, err := s.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "could not begin migration transaction")
	}
	defer s.finalizeTransaction(tx)

	migrated := 0
	for _, entry := range listEntries {
		for _, ref := range entry.refs {
			issue, ok := issueCache[ref.IssueID]
			if !ok {
				logrus.WithField("issue_id", ref.IssueID).Warn("Issue not found in KV for list reference, creating minimal record")
				issue = &kvIssue{
					ID:       ref.IssueID,
					Message:  "(migrated - original data missing)",
					CreateAt: model.GetMillis(),
				}
			}

			item := buildTodoItem(issue, entry, ref)
			_, execErr := tx.Exec(`
				INSERT INTO todo_items (
					id, user_id, list_type, message, description,
					post_permalink, post_id, status, due_date, is_pinned,
					assignee_id, tags, foreign_user_id, foreign_issue_id,
					create_at, update_at, completed_at
				) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
				ON CONFLICT (id) DO NOTHING
			`,
				item.ID, item.UserID, item.ListType, item.Message, item.Description,
				item.PostPermalink, item.PostID, item.Status, item.DueDate, item.IsPinned,
				item.AssigneeID, item.Tags, item.ForeignUserID, item.ForeignIssueID,
				item.CreateAt, item.UpdateAt, item.CompletedAt,
			)
			if execErr != nil {
				logrus.WithError(execErr).WithField("issue_id", ref.IssueID).Warn("Failed to insert migrated item")
				continue
			}
			migrated++
		}
	}

	if err := s.setSystemValue(tx, kvMigrationDoneKey, kvMigrationDoneVal); err != nil {
		return errors.Wrap(err, "failed to set KV migration done flag")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "could not commit migration transaction")
	}

	logrus.WithField("count", migrated).Info("KV->DB migration completed")
	return nil
}

type listEntry struct {
	userID   string
	listType string
	refs     []*kvIssueRef
}

func getAllKVKeys(api plugin.API) ([]string, error) {
	var allKeys []string
	page := 0
	for {
		keys, appErr := api.KVList(page, kvListPageSize)
		if appErr != nil {
			return nil, errors.New(appErr.Error())
		}
		if len(keys) == 0 {
			break
		}
		allKeys = append(allKeys, keys...)
		page++
	}
	return allKeys, nil
}

func loadKVIssue(api plugin.API, key string) (*kvIssue, error) {
	data, appErr := api.KVGet(key)
	if appErr != nil {
		return nil, errors.New(appErr.Error())
	}
	if data == nil {
		return nil, errors.New("nil data for key " + key)
	}
	var issue kvIssue
	if err := json.Unmarshal(data, &issue); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal issue")
	}
	return &issue, nil
}

func loadKVList(api plugin.API, key string) ([]*kvIssueRef, error) {
	data, appErr := api.KVGet(key)
	if appErr != nil {
		return nil, errors.New(appErr.Error())
	}
	if data == nil {
		return []*kvIssueRef{}, nil
	}

	var refs []*kvIssueRef
	if err := json.Unmarshal(data, &refs); err != nil {
		// Try legacy format (plain string array of issue IDs)
		var legacyList []string
		if legacyErr := json.Unmarshal(data, &legacyList); legacyErr != nil {
			return nil, errors.Wrap(err, "failed to unmarshal list")
		}
		refs = make([]*kvIssueRef, 0, len(legacyList))
		for _, id := range legacyList {
			refs = append(refs, &kvIssueRef{IssueID: id})
		}
	}
	return refs, nil
}

// parseListKey extracts userID and listType from a KV list key.
// Format: "order_{userID}" or "order_{userID}_in" or "order_{userID}_out"
// where userID is 26 chars.
func parseListKey(key string) (listEntry, error) {
	rest := strings.TrimPrefix(key, kvStoreListPrefix)
	if len(rest) < 26 {
		return listEntry{}, errors.New("list key too short: " + key)
	}

	userID := rest[:26]
	suffix := rest[26:]

	var listType string
	switch suffix {
	case "":
		listType = "my"
	case "_in":
		listType = "in"
	case "_out":
		listType = "out"
	default:
		return listEntry{}, errors.New("unknown list suffix: " + suffix)
	}

	return listEntry{userID: userID, listType: listType}, nil
}

func buildTodoItem(issue *kvIssue, entry listEntry, ref *kvIssueRef) *TodoItem {
	now := model.GetMillis()
	item := &TodoItem{
		ID:       issue.ID,
		UserID:   entry.userID,
		ListType: entry.listType,
		Message:  issue.Message,
		Status:   "open",
		CreateAt: issue.CreateAt,
		UpdateAt: now,
		IsPinned: false,
	}

	if issue.Description != "" {
		item.Description = &issue.Description
	}
	if issue.PostPermalink != "" {
		item.PostPermalink = &issue.PostPermalink
	}
	if issue.PostID != "" {
		item.PostID = &issue.PostID
	}
	if ref.ForeignUserID != "" {
		item.ForeignUserID = &ref.ForeignUserID
	}
	if ref.ForeignIssueID != "" {
		item.ForeignIssueID = &ref.ForeignIssueID
	}

	return item
}
