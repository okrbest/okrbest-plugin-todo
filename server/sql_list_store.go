package main

import (
	"github.com/mattermost/mattermost-plugin-todo/server/sqlstore"
	"github.com/pkg/errors"
)

// sqlListStore adapts sqlstore.SQLStore to implement the ListStore interface.
type sqlListStore struct {
	ss *sqlstore.SQLStore
}

// NewSQLListStore creates a ListStore backed by the SQL database.
func NewSQLListStore(ss *sqlstore.SQLStore) ListStore {
	return &sqlListStore{ss: ss}
}

func (s *sqlListStore) SaveIssue(issue *Issue) error {
	item := issueToTodoItem(issue)
	return s.ss.SaveTodoItem(item)
}

func (s *sqlListStore) GetIssue(issueID string) (*Issue, error) {
	item, err := s.ss.GetTodoItem(issueID)
	if err != nil {
		return nil, err
	}
	return todoItemToIssue(item), nil
}

func (s *sqlListStore) RemoveIssue(issueID string) error {
	return s.ss.DeleteTodoItem(issueID)
}

func (s *sqlListStore) GetAndRemoveIssue(issueID string) (*Issue, error) {
	item, err := s.ss.GetTodoItem(issueID)
	if err != nil {
		return nil, err
	}

	if err := s.ss.SoftDeleteTodoItem(issueID); err != nil {
		return nil, err
	}

	return todoItemToIssue(item), nil
}

func (s *sqlListStore) GetAndCompleteIssue(issueID string) (*Issue, error) {
	item, err := s.ss.GetTodoItem(issueID)
	if err != nil {
		return nil, err
	}

	if err := s.ss.CompleteTodoItem(issueID); err != nil {
		return nil, err
	}

	return todoItemToIssue(item), nil
}

func (s *sqlListStore) AddReference(userID, issueID, listID, foreignUserID, foreignIssueID string) error {
	listType := listIDToListType(listID)
	return s.ss.SetReference(issueID, userID, listType, foreignUserID, foreignIssueID)
}

func (s *sqlListStore) RemoveReference(userID, issueID, listID string) error {
	listType := listIDToListType(listID)
	return s.ss.ClearReference(issueID, userID, listType)
}

func (s *sqlListStore) PopReference(userID, listID string) (*IssueRef, error) {
	listType := listIDToListType(listID)
	item, err := s.ss.PopFirstItem(userID, listType)
	if err != nil {
		return nil, err
	}
	return todoItemToIssueRef(item), nil
}

func (s *sqlListStore) BumpReference(userID, issueID, listID string) error {
	return s.ss.SetPinned(issueID, true)
}

func (s *sqlListStore) GetIssueReference(userID, issueID, listID string) (*IssueRef, int, error) {
	listType := listIDToListType(listID)
	foreignUserID, foreignIssueID, found, err := s.ss.GetReference(userID, issueID, listType)
	if err != nil {
		return nil, 0, err
	}
	if !found {
		return nil, 0, errors.New("cannot find issue")
	}
	return &IssueRef{
		IssueID:        issueID,
		ForeignUserID:  foreignUserID,
		ForeignIssueID: foreignIssueID,
	}, 0, nil
}

func (s *sqlListStore) GetIssueListAndReference(userID, issueID string) (string, *IssueRef, int) {
	listType, foreignUserID, foreignIssueID, found, err := s.ss.FindItemList(userID, issueID)
	if err != nil || !found {
		return "", nil, 0
	}

	listID := listTypeToListID(listType)
	return listID, &IssueRef{
		IssueID:        issueID,
		ForeignUserID:  foreignUserID,
		ForeignIssueID: foreignIssueID,
	}, 0
}

func (s *sqlListStore) GetList(userID, listID string) ([]*IssueRef, error) {
	listType := listIDToListType(listID)
	items, err := s.ss.GetListItems(userID, listType)
	if err != nil {
		return nil, err
	}

	refs := make([]*IssueRef, 0, len(items))
	for _, item := range items {
		refs = append(refs, todoItemToIssueRef(item))
	}
	return refs, nil
}

// listIDToListType converts the KV list key to the SQL list_type value.
func listIDToListType(listID string) string {
	switch listID {
	case InListKey:
		return "in"
	case OutListKey:
		return "out"
	default:
		return "my"
	}
}

// listTypeToListID converts the SQL list_type back to the KV list key.
func listTypeToListID(listType string) string {
	switch listType {
	case "in":
		return InListKey
	case "out":
		return OutListKey
	default:
		return MyListKey
	}
}

func issueToTodoItem(issue *Issue) *sqlstore.TodoItem {
	item := &sqlstore.TodoItem{
		ID:       issue.ID,
		Message:  issue.Message,
		CreateAt: issue.CreateAt,
		UpdateAt: issue.UpdateAt,
		IsPinned: issue.IsPinned,
	}
	if issue.Status != "" {
		item.Status = issue.Status
	} else {
		item.Status = "open"
	}
	if issue.ListType != "" {
		item.ListType = issue.ListType
	} else {
		item.ListType = "my"
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
	if issue.DueDate != 0 {
		item.DueDate = &issue.DueDate
	}
	if issue.AssigneeID != "" {
		item.AssigneeID = &issue.AssigneeID
	}
	if issue.Tags != "" {
		item.Tags = &issue.Tags
	}
	if issue.ForeignUserID != "" {
		item.ForeignUserID = &issue.ForeignUserID
	}
	if issue.ForeignIssueID != "" {
		item.ForeignIssueID = &issue.ForeignIssueID
	}
	if issue.CompletedAt != 0 {
		item.CompletedAt = &issue.CompletedAt
	}
	return item
}

func todoItemToIssue(item *sqlstore.TodoItem) *Issue {
	issue := &Issue{
		ID:       item.ID,
		Message:  item.Message,
		CreateAt: item.CreateAt,
		UpdateAt: item.UpdateAt,
		ListType: item.ListType,
		Status:   item.Status,
		IsPinned: item.IsPinned,
	}
	if item.Description != nil {
		issue.Description = *item.Description
	}
	if item.PostPermalink != nil {
		issue.PostPermalink = *item.PostPermalink
	}
	if item.PostID != nil {
		issue.PostID = *item.PostID
	}
	if item.DueDate != nil {
		issue.DueDate = *item.DueDate
	}
	if item.AssigneeID != nil {
		issue.AssigneeID = *item.AssigneeID
	}
	if item.Tags != nil {
		issue.Tags = *item.Tags
	}
	if item.ForeignUserID != nil {
		issue.ForeignUserID = *item.ForeignUserID
	}
	if item.ForeignIssueID != nil {
		issue.ForeignIssueID = *item.ForeignIssueID
	}
	if item.CompletedAt != nil {
		issue.CompletedAt = *item.CompletedAt
	}
	return issue
}

func todoItemToIssueRef(item *sqlstore.TodoItem) *IssueRef {
	ref := &IssueRef{
		IssueID: item.ID,
	}
	if item.ForeignUserID != nil {
		ref.ForeignUserID = *item.ForeignUserID
	}
	if item.ForeignIssueID != nil {
		ref.ForeignIssueID = *item.ForeignIssueID
	}
	return ref
}
