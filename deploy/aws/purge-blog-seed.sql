-- The production blog was seeded with sample dispatches while it was being
-- built. BLOG_SEED is disabled now; remove those rows exactly once.
CREATE TABLE IF NOT EXISTS blog_maintenance (
    key TEXT PRIMARY KEY,
    done_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

WITH marker AS (
    INSERT INTO blog_maintenance (key)
    VALUES ('purge-template-dispatches-v1')
    ON CONFLICT (key) DO NOTHING
    RETURNING key
)
DELETE FROM posts
WHERE EXISTS (SELECT 1 FROM marker)
  AND (blog, slug) IN (
    ('devlog', 'a-newspaper-that-thinks-its-a-database'),
    ('devlog', 'latex-but-make-it-the-browsers-problem'),
    ('devlog', 'a-small-go-server-deliberately-boring'),
    ('devlog', 'counting-readers-without-surveillance'),
    ('devlog', 'the-cache-that-lied-at-2am'),
    ('devlog', 'epub-is-a-zip-file'),
    ('journal', 'a-desk-by-the-window'),
    ('journal', 'the-cafe-that-keeps-the-internet-at-arm''s-length'),
    ('journal', 'what-i-keep-in-the-notebook'),
    ('journal', 'a-walk-without-a-podcast'),
    ('journal', 'the-photograph-you-didnt-take'),
    ('journal', 'the-controller-on-the-shelf')
  );
