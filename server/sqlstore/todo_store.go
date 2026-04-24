package sqlstore

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"
)

// TodoItem represents a row in the todo_items table.
type TodoItem struct {
	ID             string  `db:"id"`
	UserID         string  `db:"user_id"`
	ListType       string  `db:"list_type"`
	Message        string  `db:"message"`
	Description    *string `db:"description"`
	PostPermalink  *string `db:"post_permalink"`
	PostID         *string `db:"post_id"`
	Status         string  `db:"status"`
	DueDate        *int64  `db:"due_date"`
	IsPinned       bool    `db:"is_pinned"`
	AssigneeID     *string `db:"assignee_id"`
	Tags           *string `db:"tags"`
	ForeignUserID  *string `db:"foreign_user_id"`
	ForeignIssueID *string `db:"foreign_issue_id"`
	CreateAt       int64   `db:"create_at"`
	UpdateAt       int64   `db:"update_at"`
	CompletedAt    *int64  `db:"completed_at"`
}

var todoItemColumns = []string{
	"id", "user_id", "list_type", "message", "description",
	"post_permalink", "post_id", "status", "due_date", "is_pinned",
	"assignee_id", "tags", "foreign_user_id", "foreign_issue_id",
	"create_at", "update_at", "completed_at",
}

// SaveTodoItem inserts or updates a todo item (UPSERT).
// On initial creation the row has a placeholder user_id/list_type that will be
// set by SetReference. On subsequent calls (e.g. EditIssue) only message,
// description, and update_at are overwritten to avoid clobbering ownership fields.
func (s *SQLStore) SaveTodoItem(item *TodoItem) error {
	now := model.GetMillis()
	if item.CreateAt == 0 {
		item.CreateAt = now
	}
	item.UpdateAt = now
	if item.Status == "" {
		item.Status = "open"
	}
	if item.ListType == "" {
		item.ListType = "my"
	}

	query := `
		INSERT INTO todo_items (
			id, user_id, list_type, message, description,
			post_permalink, post_id, status, due_date, is_pinned,
			assignee_id, tags, foreign_user_id, foreign_issue_id,
			create_at, update_at, completed_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT (id) DO UPDATE SET
			message      = EXCLUDED.message,
			description  = EXCLUDED.description,
			update_at    = EXCLUDED.update_at
	`

	_, err := s.exec(s.db, query,
		item.ID, item.UserID, item.ListType, item.Message, item.Description,
		item.PostPermalink, item.PostID, item.Status, item.DueDate, item.IsPinned,
		item.AssigneeID, item.Tags, item.ForeignUserID, item.ForeignIssueID,
		item.CreateAt, item.UpdateAt, item.CompletedAt,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to upsert todo item %s", item.ID)
	}

	return nil
}

// GetTodoItem retrieves a single todo item by ID.
func (s *SQLStore) GetTodoItem(id string) (*TodoItem, error) {
	var item TodoItem

	err := s.getBuilder(s.db, &item,
		sq.Select(todoItemColumns...).
			From("todo_items").
			Where(sq.Eq{"id": id}),
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("cannot find issue")
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get todo item %s", id)
	}

	return &item, nil
}

// DeleteTodoItem hard-deletes a todo item by ID.
func (s *SQLStore) DeleteTodoItem(id string) error {
	_, err := s.execBuilder(s.db,
		sq.Delete("todo_items").
			Where(sq.Eq{"id": id}),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to delete todo item %s", id)
	}
	return nil
}

// GetAndDeleteTodoItem retrieves and then hard-deletes a todo item.
func (s *SQLStore) GetAndDeleteTodoItem(id string) (*TodoItem, error) {
	item, err := s.GetTodoItem(id)
	if err != nil {
		return nil, err
	}

	if err := s.DeleteTodoItem(id); err != nil {
		return nil, err
	}

	return item, nil
}

// SetReference updates the user_id, list_type, and foreign fields on an existing todo item.
// This is the SQL equivalent of the KV AddReference operation.
func (s *SQLStore) SetReference(id, userID, listType, foreignUserID, foreignIssueID string) error {
	result, err := s.execBuilder(s.db,
		sq.Update("todo_items").
			Set("user_id", userID).
			Set("list_type", listType).
			Set("foreign_user_id", nilIfEmpty(foreignUserID)).
			Set("foreign_issue_id", nilIfEmpty(foreignIssueID)).
			Set("update_at", model.GetMillis()).
			Where(sq.Eq{"id": id}),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to set reference on todo item %s", id)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("cannot find issue to set reference")
	}

	return nil
}

// ClearReference is the SQL equivalent of RemoveReference. It sets list_type to empty
// to "unlink" the item from its current list without deleting the row. This allows
// GetAndDeleteTodoItem to still find and return the item afterward.
// Returns nil (no error) if the item was already moved to another list.
func (s *SQLStore) ClearReference(id, userID, listType string) error {
	_, err := s.execBuilder(s.db,
		sq.Update("todo_items").
			Set("list_type", "").
			Set("update_at", model.GetMillis()).
			Where(sq.Eq{"id": id, "user_id": userID, "list_type": listType}),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to clear reference on todo item %s", id)
	}
	return nil
}

// GetReference returns the foreign user/issue IDs for an open item in a specific list.
func (s *SQLStore) GetReference(userID, issueID, listType string) (foreignUserID, foreignIssueID string, found bool, err error) {
	var item TodoItem

	err = s.getBuilder(s.db, &item,
		sq.Select("foreign_user_id", "foreign_issue_id").
			From("todo_items").
			Where(sq.Eq{"id": issueID, "user_id": userID, "list_type": listType, "status": "open"}),
	)
	if err == sql.ErrNoRows {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, errors.Wrapf(err, "failed to get reference for item %s", issueID)
	}

	return ptrToString(item.ForeignUserID), ptrToString(item.ForeignIssueID), true, nil
}

// FindItemList returns which list (my/in/out) an open item belongs to for the given user.
func (s *SQLStore) FindItemList(userID, issueID string) (listType, foreignUserID, foreignIssueID string, found bool, err error) {
	var item TodoItem

	err = s.getBuilder(s.db, &item,
		sq.Select("list_type", "foreign_user_id", "foreign_issue_id").
			From("todo_items").
			Where(sq.And{
				sq.Eq{"id": issueID, "user_id": userID, "status": "open"},
				sq.NotEq{"list_type": ""},
			}),
	)
	if err == sql.ErrNoRows {
		return "", "", "", false, nil
	}
	if err != nil {
		return "", "", "", false, errors.Wrapf(err, "failed to find list for item %s", issueID)
	}

	return item.ListType, ptrToString(item.ForeignUserID), ptrToString(item.ForeignIssueID), true, nil
}

// GetListItems returns all open items for a user in a specific list, ordered by
// is_pinned DESC, due_date ASC NULLS LAST, create_at ASC.
func (s *SQLStore) GetListItems(userID, listType string) ([]*TodoItem, error) {
	var items []*TodoItem

	err := s.selectBuilder(s.db, &items,
		sq.Select(todoItemColumns...).
			From("todo_items").
			Where(sq.Eq{
				"user_id":   userID,
				"list_type": listType,
				"status":    "open",
			}).
			OrderBy("is_pinned DESC", "due_date ASC NULLS LAST", "create_at ASC"),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get list items for user %s list %s", userID, listType)
	}

	if items == nil {
		items = []*TodoItem{}
	}

	return items, nil
}

// PopFirstItem soft-deletes and returns the first open item from a user's list.
func (s *SQLStore) PopFirstItem(userID, listType string) (*TodoItem, error) {
	var item TodoItem

	err := s.getBuilder(s.db, &item,
		sq.Select(todoItemColumns...).
			From("todo_items").
			Where(sq.Eq{
				"user_id":   userID,
				"list_type": listType,
				"status":    "open",
			}).
			OrderBy("is_pinned DESC", "due_date ASC NULLS LAST", "create_at ASC").
			Limit(1),
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("cannot find issue")
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to pop item for user %s list %s", userID, listType)
	}

	if err := s.SoftDeleteTodoItem(item.ID); err != nil {
		return nil, err
	}

	return &item, nil
}

// SetPinned marks a todo item as pinned (bump to top).
func (s *SQLStore) SetPinned(id string, pinned bool) error {
	_, err := s.execBuilder(s.db,
		sq.Update("todo_items").
			Set("is_pinned", pinned).
			Set("update_at", model.GetMillis()).
			Where(sq.Eq{"id": id}),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to set pinned on todo item %s", id)
	}
	return nil
}

// UpdateMessage updates the message and description of a todo item.
func (s *SQLStore) UpdateMessage(id, message, description string) error {
	_, err := s.execBuilder(s.db,
		sq.Update("todo_items").
			Set("message", message).
			Set("description", nilIfEmpty(description)).
			Set("update_at", model.GetMillis()).
			Where(sq.Eq{"id": id}),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to update message on todo item %s", id)
	}
	return nil
}

// SoftDeleteTodoItem marks a todo item as deleted (status='deleted') without
// removing the row, preserving it for audit/dashboard analytics.
func (s *SQLStore) SoftDeleteTodoItem(id string) error {
	now := model.GetMillis()
	_, err := s.execBuilder(s.db,
		sq.Update("todo_items").
			Set("status", "deleted").
			Set("update_at", now).
			Where(sq.Eq{"id": id}),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to soft-delete todo item %s", id)
	}
	return nil
}

// CompleteTodoItem soft-deletes a todo item by setting status=done and completed_at.
func (s *SQLStore) CompleteTodoItem(id string) error {
	now := model.GetMillis()
	_, err := s.execBuilder(s.db,
		sq.Update("todo_items").
			Set("status", "done").
			Set("completed_at", now).
			Set("update_at", now).
			Where(sq.Eq{"id": id}),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to complete todo item %s", id)
	}
	return nil
}

// --- Dashboard aggregation queries ---

// TodoSummary holds the result for the personal daily summary API.
type TodoSummary struct {
	ResolvedCount int `db:"resolved_count" json:"resolved_count"`
	DelayedCount  int `db:"delayed_count" json:"delayed_count"`
}

// GetTodoSummary returns today's resolved and overdue counts for a user.
// dayStartMs/dayEndMs define the [00:00, 23:59:59.999] window in Unix ms.
func (s *SQLStore) GetTodoSummary(userID string, dayStartMs, dayEndMs int64) (*TodoSummary, error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN status='done' AND completed_at >= $2 AND completed_at <= $3 THEN 1 ELSE 0 END), 0) AS resolved_count,
			COALESCE(SUM(CASE WHEN status='open' AND due_date IS NOT NULL AND due_date < $2 THEN 1 ELSE 0 END), 0) AS delayed_count
		FROM todo_items
		WHERE user_id = $1
	`
	var summary TodoSummary
	if err := s.db.Get(&summary, query, userID, dayStartMs, dayEndMs); err != nil {
		return nil, errors.Wrapf(err, "failed to get todo summary for user %s", userID)
	}
	return &summary, nil
}

// TodoStatusSummary holds the result for the personal todo-status API.
type TodoStatusSummary struct {
	TotalIncomplete int `db:"total_incomplete" json:"total_incomplete"`
	DueToday        int `db:"due_today" json:"due_today"`
	Delayed         int `db:"delayed" json:"delayed"`
}

// GetTodoStatusSummary returns incomplete/due-today/delayed counts for a user's
// My and In lists.
func (s *SQLStore) GetTodoStatusSummary(userID string, dayStartMs, dayEndMs int64) (*TodoStatusSummary, error) {
	query := `
		SELECT
			COALESCE(SUM(1), 0) AS total_incomplete,
			COALESCE(SUM(CASE WHEN due_date >= $2 AND due_date <= $3 THEN 1 ELSE 0 END), 0) AS due_today,
			COALESCE(SUM(CASE WHEN due_date IS NOT NULL AND due_date < $2 THEN 1 ELSE 0 END), 0) AS delayed
		FROM todo_items
		WHERE user_id = $1 AND status = 'open' AND list_type IN ('my', 'in')
	`
	var summary TodoStatusSummary
	if err := s.db.Get(&summary, query, userID, dayStartMs, dayEndMs); err != nil {
		return nil, errors.Wrapf(err, "failed to get todo status summary for user %s", userID)
	}
	return &summary, nil
}

// MemberTodoStats holds per-member todo statistics for team dashboard.
type MemberTodoStats struct {
	UserID          string `db:"user_id" json:"user_id"`
	TotalCount      int    `db:"total_count" json:"total_count"`
	DoneCount       int    `db:"done_count" json:"done_count"`
	ImpendingCount  int    `db:"impending_count" json:"impending_count"`
	DelayedCount    int    `db:"delayed_count" json:"delayed_count"`
	UndefinedCount  int    `db:"undefined_count" json:"undefined_count"`
}

// GetMemberTodoStats returns todo statistics for a set of user IDs.
// doneAfterMs limits done_count to items completed after that timestamp.
// dayStartMs/dayEndMs define today's window for impending_count.
func (s *SQLStore) GetMemberTodoStats(userIDs []string, doneAfterMs, dayStartMs, dayEndMs int64) ([]*MemberTodoStats, error) {
	if len(userIDs) == 0 {
		return []*MemberTodoStats{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT
			user_id,
			COALESCE(SUM(CASE WHEN status='open' THEN 1 ELSE 0 END), 0) AS total_count,
			COALESCE(SUM(CASE WHEN status='done' AND completed_at >= ? THEN 1 ELSE 0 END), 0) AS done_count,
			COALESCE(SUM(CASE WHEN status='open' AND due_date >= ? AND due_date <= ? THEN 1 ELSE 0 END), 0) AS impending_count,
			COALESCE(SUM(CASE WHEN status='open' AND due_date IS NOT NULL AND due_date < ? THEN 1 ELSE 0 END), 0) AS delayed_count,
			COALESCE(SUM(CASE WHEN status='open' AND due_date IS NULL THEN 1 ELSE 0 END), 0) AS undefined_count
		FROM todo_items
		WHERE user_id IN (?)
		GROUP BY user_id
		ORDER BY delayed_count DESC, total_count DESC
	`, doneAfterMs, dayStartMs, dayEndMs, dayStartMs, userIDs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to expand IN clause for member todo stats")
	}

	query = s.db.Rebind(query)

	var stats []*MemberTodoStats
	if err := s.db.Select(&stats, query, args...); err != nil {
		return nil, errors.Wrap(err, "failed to get member todo stats")
	}

	if stats == nil {
		stats = []*MemberTodoStats{}
	}
	return stats, nil
}

// GetOrgResolvedCount returns the number of todo items completed within
// the given time window across the entire organisation (no user filter).
func (s *SQLStore) GetOrgResolvedCount(startMs, endMs int64) (int, error) {
	query := `
		SELECT COALESCE(COUNT(*), 0)
		FROM todo_items
		WHERE status = 'done' AND completed_at >= $1 AND completed_at <= $2
	`
	var count int
	if err := s.db.Get(&count, query, startMs, endMs); err != nil {
		return 0, errors.Wrap(err, "failed to get org resolved count")
	}
	return count, nil
}

// GetMVPRemainingCount returns the number of open todo items tagged with 'MVP'.
func (s *SQLStore) GetMVPRemainingCount() (int, error) {
	query := `
		SELECT COALESCE(COUNT(*), 0)
		FROM todo_items
		WHERE status = 'open' AND tags LIKE '%MVP%'
	`
	var count int
	if err := s.db.Get(&count, query); err != nil {
		return 0, errors.Wrap(err, "failed to get MVP remaining count")
	}
	return count, nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
