This go module is used to test the docker compose based OpenTelemetry instrumentation quickstarts:

- https://cloud.google.com/stackdriver/docs/instrumentation/setup/go
- https://cloud.google.com/stackdriver/docs/instrumentation/setup/java
- https://cloud.google.com/stackdriver/docs/instrumentation/setup/nodejs
- https://cloud.google.com/stackdriver/docs/instrumentation/setup/python


The repos hosting the quickstart code use this lib by writing a test that calls
`InstrumentationQuickstartTest()`. The tests run in a Cloud Build trigger so they can access
the Cloud Observability APIs.

`InstrumentationQuickstartTest()` runs the instrumentation quickstart docker compose setup in
the cwd and verifies that metrics, logs, and traces are successfully sent from the collector to
GCP. It checks the collector's self observability prometheus metrics to verify that the
exporters were successful.

The `COMPOSE_OVERRIDE_FILE` environment variable can be set to a comma-separated list of paths
to additional compose files to pass to docker compose (see
https://docs.docker.com/compose/multiple-compose-files/merge/). This is used in the Cloud Build
triggers to connect the docker containers to the [`cloudbuild` docker
network](https://cloud.google.com/build/docs/build-config-file-schema#network) for ADC. It can
also be used to run the test with `GOOGLE_APPLICATION_CREDENTIALS` for example.
