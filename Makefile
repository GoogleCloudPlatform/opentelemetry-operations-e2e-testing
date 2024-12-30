.PHONY: testserver-build-push
testserver-build-push:
		docker build --build-arg BUILD_TAGS=$(BUILD_TAGS) . -t opentelemetry-operations-e2e-testing:local
		docker tag opentelemetry-operations-e2e-testing:local us-central1-docker.pkg.dev/$(BUILD_PROJECT_ID)/e2e-test-runner/opentelemetry-operations-e2e-testing:ga
		docker push us-central1-docker.pkg.dev/$(BUILD_PROJECT_ID)/e2e-test-runner/opentelemetry-operations-e2e-testing:latest
.PHONY:
telemetrygenerator-build-push:
		docker build -t telgen ./telemetrygenerator
		docker tag telgen us-central1-docker.pkg.dev/ops-agent-first-dive/telemetrygenerator/telgen:ga
		docker push us-central1-docker.pkg.dev/ops-agent-first-dive/telemetrygenerator/telgen:ga



