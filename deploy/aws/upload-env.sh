#!/usr/bin/env bash
# Upload the local production env file to SSM Parameter Store.
# Run from the repository root:
#   ./deploy/aws/upload-env.sh
#   ./deploy/aws/upload-env.sh /path/to/.env.prod
set -euo pipefail

ENV_FILE="${1:-.env.prod}"
REGION="${AWS_REGION:-ap-southeast-2}"
PROFILE="${AWS_PROFILE:-osdtyp}"
PARAMETER_NAME="/osdtype/prod/env"

if [ ! -f "$ENV_FILE" ]; then
  echo "missing $ENV_FILE" >&2
  exit 1
fi

if ! git check-ignore -q "$ENV_FILE"; then
  echo "$ENV_FILE is not gitignored; refusing to upload" >&2
  exit 1
fi

if grep -q 'REPLACE_WITH_' "$ENV_FILE"; then
  echo "unresolved placeholders in $ENV_FILE:" >&2
  grep -o '^[A-Za-z_][A-Za-z0-9_]*=REPLACE_WITH_[A-Za-z0-9_]*' "$ENV_FILE" | cut -d= -f1 | sed 's/^/  /' >&2
  exit 1
fi

for key in DATABASE_URL REDIS_URL JWTKEY GITHUB_KEY GITHUB_AUTH GITHUB_REDIRECT_URL PUBLIC_ORIGIN PUBLIC_API_URL API_DOMAIN WEB_DOMAIN; do
  if ! grep -q "^${key}=..*" "$ENV_FILE"; then
    echo "missing or empty ${key} in $ENV_FILE" >&2
    exit 1
  fi
done

umask 077
aws ssm put-parameter \
  --profile "$PROFILE" \
  --region "$REGION" \
  --name "$PARAMETER_NAME" \
  --description "OSDTYP production environment" \
  --type SecureString \
  --tier Standard \
  --value "$(cat "$ENV_FILE")" \
  --overwrite >/dev/null

VERSION="$(aws ssm get-parameter \
  --profile "$PROFILE" \
  --region "$REGION" \
  --name "$PARAMETER_NAME" \
  --query 'Parameter.Version' \
  --output text)"

echo "uploaded ${PARAMETER_NAME} version ${VERSION}"
