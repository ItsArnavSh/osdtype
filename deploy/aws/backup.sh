#!/usr/bin/env bash
# Dump PostgreSQL to S3 from the EC2 host. Runs as a systemd oneshot timer.
# The instance role only has PutObject on the postgres/ prefix of this bucket.
set -euo pipefail

APP_DIR="${APP_DIR:-/opt/osdtype}"
BUCKET="${BACKUP_BUCKET:-osdtype-backups-309857583292-ap-southeast-2}"
LOCAL_DIR="${BACKUP_DIR:-/var/backups/osdtype}"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
FILE="${LOCAL_DIR}/osdtype-${STAMP}.sql.gz"

umask 077
mkdir -p "$LOCAL_DIR"
cd "$APP_DIR"

if [ ! -f .env.prod ]; then
  echo "missing ${APP_DIR}/.env.prod" >&2
  exit 1
fi

docker compose -f docker-compose.aws.yml --env-file .env.prod exec -T postgres \
  pg_dump -U osdtype -d osdtype --no-owner --no-privileges \
  | gzip -9 >"$FILE"

if [ ! -s "$FILE" ]; then
  echo "pg_dump produced an empty file" >&2
  rm -f "$FILE"
  exit 1
fi
gzip -t "$FILE"

aws s3 cp "$FILE" "s3://${BUCKET}/postgres/$(basename "$FILE")" --only-show-errors

# Keep a week of local dumps for quick restores; S3 lifecycle keeps 30 days.
find "$LOCAL_DIR" -type f -name 'osdtype-*.sql.gz' -mtime +7 -delete

echo "backup uploaded: s3://${BUCKET}/postgres/$(basename "$FILE") ($(du -h "$FILE" | cut -f1))"
