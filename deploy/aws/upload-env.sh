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

if grep -Eq '^[A-Za-z_][A-Za-z0-9_]*=.*REPLACE_WITH_' "$ENV_FILE"; then
  echo "unresolved placeholders in $ENV_FILE:" >&2
  grep -E '^[A-Za-z_][A-Za-z0-9_]*=.*REPLACE_WITH_' "$ENV_FILE" | cut -d= -f1 | sed 's/^/  /' >&2
  exit 1
fi

for key in POSTGRES_PASSWORD REDIS_PASSWORD JWTKEY GITHUB_KEY GITHUB_AUTH GITHUB_REDIRECT_URL PUBLIC_ORIGIN PUBLIC_API_URL API_DOMAIN WEB_DOMAIN; do
  if ! grep -q "^${key}=..*" "$ENV_FILE"; then
    echo "missing or empty ${key} in $ENV_FILE" >&2
    exit 1
  fi
done

for key in POSTGRES_PASSWORD REDIS_PASSWORD; do
  value="$(grep "^${key}=" "$ENV_FILE" | head -n1 | cut -d= -f2-)"
  if [ "${#value}" -lt 16 ]; then
    echo "${key} in $ENV_FILE must be at least 16 characters" >&2
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
