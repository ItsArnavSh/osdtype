
CREATE TABLE IF NOT EXISTS language_store (
    id TEXT PRIMARY KEY,
    language TEXT NOT NULL,
    snippet TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS user_data (
    user_id TEXT PRIMARY KEY,
    github_id TEXT NOT NULL UNIQUE,
    username TEXT NOT NULL,
    dp_link TEXT NOT NULL,
    top_wpm FLOAT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS type_run (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES user_data(user_id) ON DELETE CASCADE,
    language TEXT NOT NULL,
    wpm FLOAT NOT NULL,
    raw_wpm FLOAT NOT NULL,
    run_data BYTEA NOT NULL,
    delta TEXT NOT NULL,
    snippet_id TEXT NOT NULL REFERENCES language_store(id),
    start_time TIMESTAMP NOT NULL,
    is_verified BOOLEAN DEFAULT FALSE,
    is_flagged BOOLEAN DEFAULT FALSE
);
