// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

module github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing

go 1.16

require (
	cloud.google.com/go/pubsub v1.36.1
	cloud.google.com/go/storage v1.38.0
	github.com/Microsoft/go-winio v0.5.2 // indirect
	github.com/alexflint/go-arg v1.4.2
	github.com/containerd/log v0.1.0 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/docker v25.0.6+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/moby/term v0.0.0-20201216013528-df9cb8a40635 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/sethvargo/go-retry v0.1.0
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.54.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.29.0 // indirect
	golang.org/x/sync v0.8.0
	google.golang.org/api v0.169.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240822170219-fc7c04adadcd
)
