# Copyright 2022 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This Dockerfile is used to generate a docker image that runs an Apache Server.
# The Apache server is used to run a dummy default service for the Google App Engine for opentelemetry-ops-e2e project. 
# The server listens at port 8080 (since this is the port which GAE listens to).

# This docker image should currently be built manually and pushed to a proper location in the artifact repository
# Docker image name - us-central1-docker.pkg.dev/opentelemetry-ops-e2e/gae-service-containers/default-service:latest
# TODO: Automate this process

FROM httpd:2.4

# This configuration file is extracted from the above image and has been modified so that the server listens to port 8080 instead of port 80.
# The steps to do this are described here - https://hub.docker.com/_/httpd
COPY ./httpd.conf /usr/local/apache2/conf/httpd.conf
EXPOSE 8080
