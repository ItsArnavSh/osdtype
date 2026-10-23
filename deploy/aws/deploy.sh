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

# Wait for the API healthcheck before declaring success.
for _ in $(seq 1 30); do
  API_ID="$(docker compose -f docker-compose.aws.yml --env-file .env.prod ps -q api || true)"
  if [ -n "$API_ID" ]; then
    STATUS="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$API_ID" 2>/dev/null || true)"
    if [ "$STATUS" = "healthy" ] || [ "$STATUS" = "running" ]; then
      docker image prune -f >/dev/null
      echo "deployed image tag ${TAG}; api status ${STATUS}"
      exit 0
    fi
  fi
  sleep 2
done

echo "api did not become healthy" >&2
docker compose -f docker-compose.aws.yml --env-file .env.prod ps >&2 || true
docker compose -f docker-compose.aws.yml --env-file .env.prod logs --tail=120 api >&2 || true
exit 1
