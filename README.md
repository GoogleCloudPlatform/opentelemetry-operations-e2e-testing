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

## Run locally (in Google Cloud Functions)

Since running in cloud functions require you to upload a zipped file containing the source code, steps to 
trigger test runs in a cloud function environment are slightly different from running in a local environment. 

Build the docker image locally:

```bash
docker build . -t opentelemetry-operations-e2e-testing:local
```

Make sure you have the proper environment variables setup:

```bash
PROJECT_ID="opentelemetry-ops-e2e"
GOOGLE_APPLICATION_CREDENTIALS="${HOME}/.config/gcloud/application_default_credentials.json"
```

Run the image with a test server provided through the zip file. The following docker command makes the 
following assumptions: 
 - You have the source code for your function zipped in a file named `function-source.zip`
 - You have this zip file placed in a folder called `function-deployment` within your current directory *(from where you run this command)*
 - Your function code is written for `java11` environment.
    - The entrypoint for your java function is `com.google.cloud.opentelemetry.endtoend.CloudFunctionHandler`.

```bash
docker run \
    -e PROJECT_ID=${PROJECT_ID} \
    -e GOOGLE_APPLICATION_CREDENTIALS=${GOOGLE_APPLICATION_CREDENTIALS} \
    -v "${GOOGLE_APPLICATION_CREDENTIALS}:${GOOGLE_APPLICATION_CREDENTIALS}:ro" \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v "$(pwd)/function-deployment/function-source.zip:/function-source.zip" \ 
    -it \
    --rm \
    opentelemetry-operations-e2e-testing:local \
    cloud-functions-gen2 \
    --runtime=java11 \
    --functionsource="/function-source.zip" \
    --entrypoint=com.google.cloud.opentelemetry.endtoend.CloudFunctionHandler
```

NOTE: If you are using Java, you can also use a generated JAR to deploy the function. However, you would still need zip the JAR and the JAR should be at the root of the zip. For more information look at the [How to guides](https://cloud.google.com/functions/docs/how-to) for Google Cloud Functions.

## [Matrix of implemented scenarios](matrix.md)

## Contributing

See [`docs/contributing.md`](docs/contributing.md) for details.

## License

Apache 2.0; see [`LICENSE`](LICENSE) for details.

## Disclaimer

This project is not an official Google project. It is not supported by
Google and Google specifically disclaims all warranties as to its quality,
merchantability, or fitness for a particular purpose.
