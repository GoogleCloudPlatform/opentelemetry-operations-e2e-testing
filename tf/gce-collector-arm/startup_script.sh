#!/bin/bash

export HOME=/home/appuser

mkdir /tmp/config

cat > /tmp/config/config.json <<EOF
${config}
EOF

# Configure docker with credentials for gcr.io and pkg.dev
docker-credential-gcr configure-docker --registries us-docker.pkg.dev

sudo -E docker run -v /tmp/config:/config "${image}" --config=/config/config.json
