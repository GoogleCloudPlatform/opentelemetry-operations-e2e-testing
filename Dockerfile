# Copyright 2021 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.23 AS gobuild
WORKDIR /src
ARG BUILD_TAGS

# cache deps before copying source so that we don't re-download as much
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN set -x; if [ "$BUILD_TAGS" = "e2ecollector" ]; then BUILD_DIRECTORY="e2etestrunner_collector"; else BUILD_TAGS="e2e"; BUILD_DIRECTORY="e2etestrunner"; fi; CGO_ENABLED=0 go test -timeout 3600s -tags=$BUILD_TAGS -c "./$BUILD_DIRECTORY" -o opentelemetry-operations-e2e-testing.test

FROM hashicorp/terraform:1.13 as tfbuild
COPY tf /src/tf
WORKDIR /src/tf
ENV TF_PLUGIN_CACHE_DIR=/src/tf/terraform-plugin-cache
# run terraform init in each directory to cache the modules
RUN for dir in */; do (cd $dir && terraform init -backend=false); done

FROM alpine:3.14
RUN apk --update add ca-certificates git
COPY --from=gobuild /src/opentelemetry-operations-e2e-testing.test /opentelemetry-operations-e2e-testing.test
COPY --from=tfbuild /src/tf /tf
COPY --from=tfbuild /bin/terraform /bin/terraform
ENV TF_PLUGIN_CACHE_DIR=/tf/terraform-plugin-cache
ENTRYPOINT ["/opentelemetry-operations-e2e-testing.test", "--gotestflags=-test.v"]
