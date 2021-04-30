#!/bin/bash

GOOGLE_APPLICATION_CREDENTIALS=$HOME/.config/gcloud/application_default_credentials.json

docker run \
    -e GOOGLE_APPLICATION_CREDENTIALS=$GOOGLE_APPLICATION_CREDENTIALS \
    -v "$GOOGLE_APPLICATION_CREDENTIALS:$GOOGLE_APPLICATION_CREDENTIALS:ro" \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -e PROJECT_ID=$PROJECT_ID \
    --rm \
    opentelemetry-operations-e2e-testing:local \
    local \
    --image=operations-python-e2e-test-server:0.1.4 \
    --gotestflags=-test.v
