# opentelemetry-operations-e2e-testing

This project contains a test runner and configuration which are used for testing
GCP's OpenTelemetry exporters, resource detectors, propagators, etc. These tests
are run in pull requests to the other
[GoogleCloudPlatform/opentelemtry-operations-*](https://github.com/GoogleCloudPlatform/?q=opentelemetry-operations-&type=&language=&sort=)
repos.

The code in this repository is only for testing purposes! This project is not an
official Google project. It is not supported by Google and Google specifically
disclaims all warranties as to its quality, merchantability, or fitness for a
particular purpose.

## Run locally

Build the docker image locally:

```bash
docker build . -t opentelemetry-operations-e2e-testing:local
```

Run the image with a test server, e.g. with the python instrumented test server
from a recent build:

```bash
PROJECT_ID="opentelemetry-ops-e2e"
GOOGLE_APPLICATION_CREDENTIALS="${HOME}/.config/gcloud/application_default_credentials.json"

# Using a recent python build for example. Alternatively, use a locally built
# test server.
INSTRUMENTED_TEST_SERVER="gcr.io/opentelemetry-ops-e2e/opentelemetry-operations-python-e2e-test-server:45ccd1d"

# Pull the image if it doesn't exist locally
docker pull ${INSTRUMENTED_TEST_SERVER}
docker run \
    -e "GOOGLE_APPLICATION_CREDENTIALS=${GOOGLE_APPLICATION_CREDENTIALS}" \
    -v "${GOOGLE_APPLICATION_CREDENTIALS}:${GOOGLE_APPLICATION_CREDENTIALS}:ro" \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -e PROJECT_ID=${PROJECT_ID} \
    --rm \
    opentelemetry-operations-e2e-testing:local \
    local \
    --image=${INSTRUMENTED_TEST_SERVER}
```

## [Matrix of implemented scenarios](matrix.md)

## Contributing

See [`docs/contributing.md`](docs/contributing.md) for details.

## License

Apache 2.0; see [`LICENSE`](LICENSE) for details.

## Disclaimer

This project is not an official Google project. It is not supported by
Google and Google specifically disclaims all warranties as to its quality,
merchantability, or fitness for a particular purpose.
