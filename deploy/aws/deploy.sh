#!/usr/bin/env bash
# Runs on the EC2 instance, normally via SSM Run Command.
# Usage: deploy.sh <image-tag>
set -euxo pipefail

TAG="${1:-latest}"
REGION="ap-southeast-2"
PARAMETER_NAME="/osdtype/prod/env"
APP_DIR="/opt/osdtype"

cd "$APP_DIR"

# Fetch the real environment from SSM. Never print this file.
umask 077
aws ssm get-parameter \
  --region "$REGION" \
  --name "$PARAMETER_NAME" \
  --with-decryption \
  --query 'Parameter.Value' \
  --output text >.env.prod.tmp
mv .env.prod.tmp .env.prod
chmod 600 .env.prod

# Pull the latest compose/Caddy deployment files.
git fetch --depth=1 origin main
git reset --hard origin/main

ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"
export ECR_REGISTRY="${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com"
export IMAGE_TAG="$TAG"
export BLOG_IMAGE_TAG="blog-${TAG}"

aws ecr get-login-password --region "$REGION" \
  | docker login --username AWS --password-stdin "$ECR_REGISTRY"

docker compose -f docker-compose.aws.yml --env-file .env.prod pull
docker compose -f docker-compose.aws.yml --env-file .env.prod up -d --remove-orphans

# Keep the daily Postgres backup timer installed and enabled.
chmod +x deploy/aws/backup.sh
install -m 0644 deploy/aws/osdtype-backup.service /etc/systemd/system/osdtype-backup.service
install -m 0644 deploy/aws/osdtype-backup.timer /etc/systemd/system/osdtype-backup.timer
systemctl daemon-reload
systemctl enable --now osdtype-backup.timer

# Wait for both the API and the blog before declaring success.
API_STATUS=""
BLOG_STATUS=""
for _ in $(seq 1 60); do
  API_ID="$(docker compose -f docker-compose.aws.yml --env-file .env.prod ps -q api || true)"
  BLOG_ID="$(docker compose -f docker-compose.aws.yml --env-file .env.prod ps -q blog || true)"
  if [ -n "$API_ID" ]; then
    API_STATUS="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$API_ID" 2>/dev/null || true)"
  fi
  if [ -n "$BLOG_ID" ]; then
    BLOG_STATUS="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$BLOG_ID" 2>/dev/null || true)"
  fi
  if { [ "$API_STATUS" = "healthy" ] || [ "$API_STATUS" = "running" ]; } && [ "$BLOG_STATUS" = "healthy" ]; then
    # Remove the sample dispatches once. The marker table makes this a no-op
    # on later deploys.
    docker compose -f docker-compose.aws.yml --env-file .env.prod exec -T postgres \
      psql -U osdtype -d arnav_blog -v ON_ERROR_STOP=1 <deploy/aws/purge-blog-seed.sql

    docker image prune -f >/dev/null
    echo "deployed image tag ${TAG}; api status ${API_STATUS}; blog status ${BLOG_STATUS}"
    exit 0
  fi
  sleep 2
done

echo "services did not become healthy (api=${API_STATUS:-missing}, blog=${BLOG_STATUS:-missing})" >&2
docker compose -f docker-compose.aws.yml --env-file .env.prod ps >&2 || true
docker compose -f docker-compose.aws.yml --env-file .env.prod logs --tail=120 api blog >&2 || true
exit 1
