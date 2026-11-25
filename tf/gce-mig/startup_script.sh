#!/bin/bash
set -euo pipefail

export DOCKER_CONFIG=/tmp/docker-config
mkdir -p /tmp/docker-config

# Configure docker with credentials for Artifact Registry and GCR
docker-credential-gcr configure-docker --registries "${registries}"

docker run \
  -e "PROJECT_ID=${project_id}" \
  -e "REQUEST_SUBSCRIPTION_NAME=${request_subscription_name}" \
  -e "RESPONSE_TOPIC_NAME=${response_topic_name}" \
  -e "SUBSCRIPTION_MODE=${subscription_mode}" \
  "${image}"
