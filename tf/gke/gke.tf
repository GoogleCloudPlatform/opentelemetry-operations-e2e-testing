# Copyright 2021 Google
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

resource "google_container_cluster" "default" {
  # The terraform workspace will be given a random name (test run id) which we
  # can use to get unique resource names.
  name = "e2etest-${terraform.workspace}"
  location           = "us-central1"

  initial_node_count = 1
  node_config {
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
  }
}

data "google_client_config" "default" {}
provider "kubernetes" {
  host = "https://${google_container_cluster.default.endpoint}"
  cluster_ca_certificate = "${base64decode(google_container_cluster.default.master_auth.0.cluster_ca_certificate)}"
  token = "${data.google_client_config.default.access_token}"
}

resource "kubernetes_pod" "testserver" {
  metadata {
    name = "testserver-${terraform.workspace}"
  }

  spec {
    container {
      image = var.image
      name  = "testserver-${terraform.workspace}-container"

      env {
        name  = "PROJECT_ID"
        value = var.project_id
      }
      env {
        name = "REQUEST_SUBSCRIPTION_NAME"
        value = module.pubsub.info.request_topic.subscription_name
      }
      env {
        name = "RESPONSE_TOPIC_NAME"
        value = module.pubsub.info.response_topic.topic_name
      }
    }
  }
}

module "pubsub" {
  source = "../modules/pubsub"

  project_id = var.project_id
}

variable "image" {
  type = string
}

output "pubsub_info" {
  value = module.pubsub.info
  description = "Info about the request/response pubsub topics and subscription to use in the test"
}
