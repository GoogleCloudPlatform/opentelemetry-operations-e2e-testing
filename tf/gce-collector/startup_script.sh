#!/bin/bash
set -euo pipefail

export DOCKER_CONFIG=/tmp/docker-config
mkdir -p /tmp/docker-config /tmp/config

cat > /tmp/config/config.json <<'EOF'
${config}
EOF

# Configure docker with credentials for Artifact Registry and GCR
docker-credential-gcr configure-docker --registries "${registries}"

docker run -v /tmp/config:/config "${image}" --config=/config/config.json
