
-- =====================
-- language_store queries
-- =====================

-- name: UpsertLanguageSnippet :exec
INSERT INTO language_store (id, language, snippet)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE
SET language = EXCLUDED.language,
    snippet = EXCLUDED.snippet;

-- name: GetSnippetByID :one
SELECT id, language, snippet
FROM language_store
WHERE id = $1;


-- name: GetRandomSnippetByLanguage :one
SELECT id, language, snippet
FROM language_store
WHERE language = $1
ORDER BY RANDOM()
LIMIT 1;

-- =====================
-- user_data queries
-- =====================

-- name: UpsertUser :exec
INSERT INTO user_data (user_id, github_id, username, dp_link, top_wpm)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id) DO UPDATE
SET username = EXCLUDED.username,
    dp_link = EXCLUDED.dp_link,
    top_wpm = GREATEST(user_data.top_wpm, EXCLUDED.top_wpm);

-- name: GetUserByID :one
SELECT user_id, github_id, username, dp_link, top_wpm
FROM user_data
WHERE user_id = $1;

-- name: GetUserByGithubID :one
SELECT user_id, github_id, username, dp_link, top_wpm
FROM user_data
WHERE github_id = $1;

-- name: UpdateTopWPM :exec
UPDATE user_data
SET top_wpm = GREATEST(top_wpm, $2)
WHERE user_id = $1;

-- =====================
-- type_run queries
-- =====================

-- name: CreateTypeRun :exec
INSERT INTO type_run (
    id, user_id, language, wpm, raw_wpm,
    run_data, delta, snippet_id, start_time
)
VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9
);

-- name: GetRunsByUser :many
SELECT *
FROM type_run
WHERE user_id = $1
ORDER BY start_time DESC;

-- name: GetTypeRunByID :one
SELECT *
FROM type_run
WHERE id = $1;

-- name: VerifyTypeRun :exec
UPDATE type_run
SET is_verified = TRUE
WHERE id = $1;

-- name: FlagTypeRun :exec
UPDATE type_run
SET is_flagged = TRUE
WHERE id = $1;

-- name: GetGlobalLeaderboard :many
SELECT tr.id, tr.user_id, u.username, u.dp_link, tr.wpm, tr.language, tr.start_time
FROM type_run tr
JOIN user_data u ON tr.user_id = u.user_id
WHERE tr.is_verified = TRUE AND tr.is_flagged = FALSE
ORDER BY tr.wpm DESC
LIMIT $1;

-- name: GetLanguageLeaderboard :many
SELECT tr.id, tr.user_id, u.username, u.dp_link, tr.wpm, tr.start_time
FROM type_run tr
JOIN user_data u ON tr.user_id = u.user_id
WHERE tr.language = $1 AND tr.is_verified = TRUE AND tr.is_flagged = FALSE
ORDER BY tr.wpm DESC
LIMIT $2;
