# Copyright 2021 Google
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

FROM golang:1.16 AS build
WORKDIR /src

COPY . .
RUN CGO_ENABLED=0 go test -c
# need a dummy file to create /tmp dir in the scratch image
RUN touch /.empty

FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
COPY --from=build /src/opentelemetry-operations-e2e-testing.test /opentelemetry-operations-e2e-testing.test
COPY --from=build /src/tf /tf
COPY --from=build /.empty /tmp/
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=hashicorp/terraform:light /bin/terraform /bin/terraform
ENTRYPOINT ["/opentelemetry-operations-e2e-testing.test", "--gotestflags=-test.v"]
