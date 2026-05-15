CREATE TABLE IF NOT EXISTS TODO_System (
    SKey   VARCHAR(64) PRIMARY KEY,
    SValue VARCHAR(1024) NULL
);

CREATE TABLE IF NOT EXISTS todo_items (
    id               VARCHAR(26) PRIMARY KEY,
    user_id          VARCHAR(26) NOT NULL,
    list_type        VARCHAR(10) NOT NULL DEFAULT 'my',
    message          TEXT NOT NULL,
    description      TEXT,
    post_permalink   VARCHAR(512),
    post_id          VARCHAR(26),
    status           VARCHAR(20) NOT NULL DEFAULT 'open',
    due_date         BIGINT,
    is_pinned        BOOLEAN NOT NULL DEFAULT false,
    assignee_id      VARCHAR(26),
    tags             TEXT,
    foreign_user_id  VARCHAR(26),
    foreign_issue_id VARCHAR(26),
    create_at        BIGINT NOT NULL,
    update_at        BIGINT NOT NULL,
    completed_at     BIGINT
);

CREATE INDEX IF NOT EXISTS idx_todo_items_user_status
    ON todo_items(user_id, status);

CREATE INDEX IF NOT EXISTS idx_todo_items_user_list
    ON todo_items(user_id, list_type);

CREATE INDEX IF NOT EXISTS idx_todo_items_user_due
    ON todo_items(user_id, due_date) WHERE status = 'open';

CREATE INDEX IF NOT EXISTS idx_todo_items_completed_at
    ON todo_items(user_id, completed_at) WHERE status = 'done';

CREATE INDEX IF NOT EXISTS idx_todo_items_assignee
    ON todo_items(assignee_id, status);
