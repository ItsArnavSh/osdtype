-- The production blog was seeded with sample dispatches while it was being
-- built. Remove those rows. This is safe to run on every deploy: the slugs
-- belong to the retired sample editions, and the marker records the cleanup.
CREATE TABLE IF NOT EXISTS blog_maintenance (
    key TEXT PRIMARY KEY,
    done_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DELETE FROM posts
WHERE (blog, slug) IN (
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

INSERT INTO blog_maintenance (key)
VALUES ('purge-template-dispatches-v1')
ON CONFLICT (key) DO NOTHING;
