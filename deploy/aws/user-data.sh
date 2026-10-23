#!/usr/bin/env bash
# EC2 first-boot setup for OSDTYP. No secrets are stored here.
set -euxo pipefail

dnf install -y docker git unzip
systemctl enable --now docker
usermod -aG docker ec2-user

# Keep container logs from filling the small root volume.
mkdir -p /etc/docker
cat >/etc/docker/daemon.json <<'EOF'
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
EOF
systemctl restart docker

# 1 GiB swap gives the Go runtime and image pulls headroom on a 1 GiB box.
if [ ! -f /swapfile ]; then
  fallocate -l 1G /swapfile || dd if=/dev/zero of=/swapfile bs=1M count=1024
  chmod 600 /swapfile
  mkswap /swapfile
  swapon /swapfile
  echo '/swapfile none swap sw 0 0' >>/etc/fstab
fi
printf 'vm.swappiness=10\n' >/etc/sysctl.d/99-swappiness.conf
sysctl --system

# AL2023 usually ships the SSM agent. Start it in case it is present but disabled.
systemctl enable --now amazon-ssm-agent || true

# Ensure the AWS CLI is available for ECR login and SSM parameter retrieval.
if ! command -v aws >/dev/null 2>&1; then
  ARCH="$(uname -m)"
  curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-${ARCH}.zip" -o /tmp/awscliv2.zip
  unzip -q /tmp/awscliv2.zip -d /tmp
  /tmp/aws/install
fi

# Docker Compose plugin (AL2023's docker package does not guarantee it).
COMPOSE_VERSION="v5.6.1"
ARCH="$(uname -m)"
mkdir -p /usr/local/lib/docker/cli-plugins
curl -fsSL "https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-linux-${ARCH}" \
  -o /usr/local/lib/docker/cli-plugins/docker-compose
curl -fsSL "https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-linux-${ARCH}.sha256" \
  -o /tmp/docker-compose.sha256
EXPECTED="$(cut -d' ' -f1 /tmp/docker-compose.sha256)"
ACTUAL="$(sha256sum /usr/local/lib/docker/cli-plugins/docker-compose | cut -d' ' -f1)"
if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "docker compose checksum mismatch" >&2
  exit 1
fi
chmod +x /usr/local/lib/docker/cli-plugins/docker-compose

# The deploy script fetches the real environment from SSM Parameter Store.
if [ ! -d /opt/osdtype/.git ]; then
  for attempt in 1 2 3 4 5; do
    if git clone --depth=1 https://github.com/ItsArnavSh/osdtype.git /opt/osdtype; then
      break
    fi
    rm -rf /opt/osdtype
    sleep 5
  done
fi
git config --system --add safe.directory /opt/osdtype
chown -R ec2-user:ec2-user /opt/osdtype

docker compose version
echo "OSDTYP first-boot setup complete"
