#!/usr/bin/env bash
# Create the CloudFront + AWS WAF edge for the OSDTYP API.
#
# Run from the repository root after .env.prod contains ORIGIN_VERIFY_SECRET:
#   ./deploy/aws/setup-edge.sh
#
# The script creates an ACM certificate in us-east-1, prints the DNS validation
# record, waits for issuance, deploys the CloudFormation stack, and prints the
# final CNAME target for api.osd-typ.in. It does NOT switch the API URL or
# enable origin verification; do that only after the DNS record resolves.
set -euo pipefail

REGION="${AWS_REGION:-us-east-1}"
PROFILE="${AWS_PROFILE:-osdtyp}"
STACK_NAME="${STACK_NAME:-osdtype-edge}"
API_DOMAIN="${API_DOMAIN:-api.osd-typ.in}"
# A wildcard cert avoids a CNAME conflict: ACM validates *.osd-typ.in at
# _token.osd-typ.in, which can coexist with the api.osd-typ.in CNAME.
CERT_DOMAIN="${CERT_DOMAIN:-*.${API_DOMAIN#*.}}"
ORIGIN_DOMAIN="${ORIGIN_DOMAIN:-api.15.134.14.153.sslip.io}"
ENV_FILE="${ENV_FILE:-.env.prod}"
TEMPLATE="deploy/aws/cloudfront-waf.yaml"
PRICE_CLASS="${PRICE_CLASS:-PriceClass_200}"
LOG_RETENTION_DAYS="${LOG_RETENTION_DAYS:-30}"

AWS=(aws --profile "$PROFILE" --region "$REGION")

if [ ! -f "$ENV_FILE" ]; then
  echo "missing $ENV_FILE" >&2
  exit 1
fi
if [ ! -f "$TEMPLATE" ]; then
  echo "missing $TEMPLATE; run from the repository root" >&2
  exit 1
fi

ORIGIN_VERIFY_SECRET="$(grep '^ORIGIN_VERIFY_SECRET=' "$ENV_FILE" | head -n1 | cut -d= -f2-)"
if [ "${#ORIGIN_VERIFY_SECRET}" -lt 32 ]; then
  echo "ORIGIN_VERIFY_SECRET in $ENV_FILE must be at least 32 characters" >&2
  exit 1
fi

echo "==> Checking for an existing certificate for ${CERT_DOMAIN}"
CERT_ARN="$("${AWS[@]}" acm list-certificates \
  --query "CertificateSummaryList[?DomainName=='${CERT_DOMAIN}'].CertificateArn | [0]" \
  --output text 2>/dev/null || true)"
if [ -z "$CERT_ARN" ] || [ "$CERT_ARN" = "None" ]; then
  echo "==> Requesting a DNS-validated certificate for ${CERT_DOMAIN} in us-east-1"
  CERT_ARN="$("${AWS[@]}" acm request-certificate \
    --domain-name "$CERT_DOMAIN" \
    --validation-method DNS \
    --idempotency-token "osdtype-native-api" \
    --query 'CertificateArn' --output text)"
else
  echo "    reusing ${CERT_ARN}"
fi

# The validation record can take a few seconds to appear.
VALIDATION_NAME=""
VALIDATION_VALUE=""
for _ in $(seq 1 30); do
  VALIDATION="$("${AWS[@]}" acm describe-certificate \
    --certificate-arn "$CERT_ARN" \
    --query 'Certificate.DomainValidationOptions[0].ResourceRecord.[Name,Value]' \
    --output text 2>/dev/null || true)"
  if [ -n "$VALIDATION" ]; then
    VALIDATION_NAME="$(printf '%s\n' "$VALIDATION" | awk '{print $1}')"
    VALIDATION_VALUE="$(printf '%s\n' "$VALIDATION" | awk '{print $2}')"
  fi
  if [ -n "$VALIDATION_NAME" ] && [ -n "$VALIDATION_VALUE" ]; then
    break
  fi
  sleep 2
done

if [ -z "$VALIDATION_NAME" ] || [ -z "$VALIDATION_VALUE" ]; then
  echo "could not read the ACM DNS validation record" >&2
  exit 1
fi

VALIDATION_NAME="${VALIDATION_NAME%.}"
VALIDATION_VALUE="${VALIDATION_VALUE%.}"

cat <<EOF

==> Add this DNS record at GoDaddy (or your DNS provider):

    Type:  CNAME
    Name:  ${VALIDATION_NAME}
    Value: ${VALIDATION_VALUE}

    GoDaddy's Name field often omits the domain suffix; if it appends
    ${API_DOMAIN}, enter only the part before .${API_DOMAIN}.

EOF

STATUS="$("${AWS[@]}" acm describe-certificate --certificate-arn "$CERT_ARN" \
  --query 'Certificate.Status' --output text)"
if [ "$STATUS" != "ISSUED" ]; then
  echo "==> Waiting for ACM validation (up to 45 minutes)"
  for _ in $(seq 1 135); do
    STATUS="$("${AWS[@]}" acm describe-certificate --certificate-arn "$CERT_ARN" \
      --query 'Certificate.Status' --output text)"
    case "$STATUS" in
      ISSUED) break ;;
      FAILED|VALIDATION_TIMED_OUT|REVOKED)
        echo "certificate status is ${STATUS}" >&2
        exit 1
        ;;
    esac
    sleep 20
  done
fi
if [ "$STATUS" != "ISSUED" ]; then
  echo "certificate was not issued yet (status: ${STATUS}); re-run this script when the DNS record is live" >&2
  exit 1
fi

echo "==> Deploying ${STACK_NAME} in us-east-1"
"${AWS[@]}" cloudformation deploy \
  --stack-name "$STACK_NAME" \
  --template-file "$TEMPLATE" \
  --no-fail-on-empty-changeset \
  --parameter-overrides \
    "ApiDomainName=${API_DOMAIN}" \
    "OriginDomainName=${ORIGIN_DOMAIN}" \
    "CertificateArn=${CERT_ARN}" \
    "OriginVerifySecret=${ORIGIN_VERIFY_SECRET}" \
    "PriceClass=${PRICE_CLASS}" \
    "WafLogRetentionDays=${LOG_RETENTION_DAYS}" \
  --tags Application=osdtype

DIST_DOMAIN="$("${AWS[@]}" cloudformation describe-stacks --stack-name "$STACK_NAME" \
  --query "Stacks[0].Outputs[?OutputKey=='DistributionDomainName'].OutputValue" \
  --output text)"
DIST_ID="$("${AWS[@]}" cloudformation describe-stacks --stack-name "$STACK_NAME" \
  --query "Stacks[0].Outputs[?OutputKey=='DistributionId'].OutputValue" \
  --output text)"

cat <<EOF

==> CloudFront is ready.

    Distribution ID:  ${DIST_ID}
    Target hostname:  ${DIST_DOMAIN}

==> Add this second DNS record at GoDaddy:

    Type:  CNAME
    Name:  api
    Value: ${DIST_DOMAIN}

==> Wait for DNS, then verify through the edge:

    dig +short ${API_DOMAIN}
    curl -fsS https://${API_DOMAIN}/ping

==> After /ping works, complete the cutover:
    1. Set PUBLIC_API_URL=https://${API_DOMAIN} and
       GITHUB_REDIRECT_URL=https://${API_DOMAIN}/auth/github/callback
       in .env.prod, then run ./deploy/aws/upload-env.sh.
    2. Update the GitHub OAuth app callback to
       https://${API_DOMAIN}/auth/github/callback.
    3. Set VITE_API_URL=https://${API_DOMAIN} in Vercel and redeploy the frontend.
    4. Run ./deploy/aws/deploy.sh latest on the instance.
    5. Verify sign-in, then set ORIGIN_VERIFY_ENABLED=true in .env.prod,
       upload, and deploy again to lock the origin to CloudFront.
EOF
