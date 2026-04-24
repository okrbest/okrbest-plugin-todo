package sqlstore

import (
	"github.com/blang/semver"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// Migration defines a schema migration with a version range and a function to execute.
type Migration struct {
	fromVersion   semver.Version
	toVersion     semver.Version
	migrationFunc func(sqlx.Ext, *SQLStore) error
}

var migrations = []Migration{
	{
		fromVersion: semver.MustParse("0.0.0"),
		toVersion:   semver.MustParse("0.1.0"),
		migrationFunc: func(e sqlx.Ext, sqlStore *SQLStore) error {
			if _, err := e.Exec(`
				CREATE TABLE IF NOT EXISTS TODO_System (
					SKey  VARCHAR(64) PRIMARY KEY,
					SValue VARCHAR(1024) NULL
				);
			`); err != nil {
				return errors.Wrapf(err, "failed creating table TODO_System")
			}

			if _, err := e.Exec(`
				CREATE TABLE IF NOT EXISTS todo_items (
					id              VARCHAR(26) PRIMARY KEY,
					user_id         VARCHAR(26) NOT NULL,
					list_type       VARCHAR(10) NOT NULL DEFAULT 'my',
					message         TEXT NOT NULL,
					description     TEXT,
					post_permalink  VARCHAR(512),
					post_id         VARCHAR(26),
					status          VARCHAR(20) NOT NULL DEFAULT 'open',
					due_date        BIGINT,
					is_pinned       BOOLEAN NOT NULL DEFAULT false,
					assignee_id     VARCHAR(26),
					tags            TEXT,
					foreign_user_id VARCHAR(26),
					foreign_issue_id VARCHAR(26),
					create_at       BIGINT NOT NULL,
					update_at       BIGINT NOT NULL,
					completed_at    BIGINT
				);
			`); err != nil {
				return errors.Wrapf(err, "failed creating table todo_items")
			}

			if _, err := e.Exec(`
				CREATE INDEX IF NOT EXISTS idx_todo_items_user_status
					ON todo_items(user_id, status);
			`); err != nil {
				return errors.Wrapf(err, "failed creating index idx_todo_items_user_status")
			}

			if _, err := e.Exec(`
				CREATE INDEX IF NOT EXISTS idx_todo_items_user_list
					ON todo_items(user_id, list_type);
			`); err != nil {
				return errors.Wrapf(err, "failed creating index idx_todo_items_user_list")
			}

			if _, err := e.Exec(`
				CREATE INDEX IF NOT EXISTS idx_todo_items_user_due
					ON todo_items(user_id, due_date) WHERE status = 'open';
			`); err != nil {
				return errors.Wrapf(err, "failed creating index idx_todo_items_user_due")
			}

			if _, err := e.Exec(`
				CREATE INDEX IF NOT EXISTS idx_todo_items_completed_at
					ON todo_items(user_id, completed_at) WHERE status = 'done';
			`); err != nil {
				return errors.Wrapf(err, "failed creating index idx_todo_items_completed_at")
			}

			if _, err := e.Exec(`
				CREATE INDEX IF NOT EXISTS idx_todo_items_assignee
					ON todo_items(assignee_id, status);
			`); err != nil {
				return errors.Wrapf(err, "failed creating index idx_todo_items_assignee")
			}

			return nil
		},
	},
}
