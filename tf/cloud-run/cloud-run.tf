# Copyright 2021 Google LLC
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

resource "google_cloud_run_service" "default" {
  name     = "e2etest-${terraform.workspace}"
  location = "us-central1"

  timeouts {
    create = "1m"
  }

  template {
    metadata {
      labels = local.common_labels
      annotations = {
        // limit to one container instance
        "autoscaling.knative.dev/maxScale" = "1"
      }
    }

    spec {
      containers {
        image = var.image

        ports {
          name           = "http1"
          container_port = local.push_port
        }

        env {
          name  = "PROJECT_ID"
          value = var.project_id
        }
        env {
          name  = "REQUEST_SUBSCRIPTION_NAME"
          value = module.pubsub.info.request_topic.subscription_name
        }
        env {
          name  = "RESPONSE_TOPIC_NAME"
          value = module.pubsub.info.response_topic.topic_name
        }
        env {
          name  = "SUBSCRIPTION_MODE"
          value = "push"
        }
        env {
          name  = "PUSH_PORT"
          value = local.push_port
        }
      }
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }
}

module "pubsub" {
  source = "../modules/pubsub"

  project_id = var.project_id
}

module "pubsub-push-subscription" {
  source = "../modules/pubsub-push-subscription"

  project_id    = var.project_id
  topic         = module.pubsub.info.request_topic.topic_name
  push_endpoint = google_cloud_run_service.default.status[0].url
}

locals {
  push_port = "8000"
}

variable "image" {
  type = string
}

output "pubsub_info" {
  value       = module.pubsub.info
  description = "Info about the request/response pubsub topics and subscription to use in the test"
}
