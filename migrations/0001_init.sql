CREATE TABLE IF NOT EXISTS teams (
    id        BIGSERIAL PRIMARY KEY,
    team_name TEXT NOT NULL UNIQUE 
);

CREATE TABLE IF NOT EXISTS users (
    user_id   TEXT PRIMARY KEY,
    username  TEXT NOT NULL,
    team_id   BIGINT NOT NULL REFERENCES teams(id) ON DELETE RESTRICT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TYPE pr_status AS ENUM ('OPEN', 'MERGED');


CREATE TABLE IF NOT EXISTS pull_requests (
    pull_request_id   TEXT PRIMARY KEY,
    pull_request_name TEXT NOT NULL,
    author_id         TEXT NOT NULL REFERENCES users(user_id) ON DELETE RESTRICT,
    status            pr_status NOT NULL DEFAULT 'OPEN',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    merged_at         TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS pull_request_reviewers (
    pull_request_id TEXT NOT NULL REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
    reviewer_id     TEXT NOT NULL REFERENCES users(user_id) ON DELETE RESTRICT,
    PRIMARY KEY (pull_request_id, reviewer_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_team_username ON users(team_id, username);
CREATE INDEX IF NOT EXISTS idx_users_team_is_active ON users(team_id, is_active);
CREATE INDEX IF NOT EXISTS idx_pr_status ON pull_requests(status);
CREATE INDEX IF NOT EXISTS idx_pr_rev_reviewer ON pull_request_reviewers(reviewer_id);
CREATE INDEX IF NOT EXISTS idx_pr_author ON pull_requests(author_id);
