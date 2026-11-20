# Copyright 2026 Google LLC
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

resource "google_cloud_run_v2_instance" "default" {
  name                 = "e2etest-inst-${terraform.workspace}"
  location             = var.location
  default_uri_disabled = true

  labels = local.common_labels

  timeouts {
    create = "6m"
  }

  containers {
    image = var.image

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
      value = "pull"
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

variable "location" {
  type    = string
  default = "us-central1"
}

output "pubsub_info" {
  value       = module.pubsub.info
  description = "Info about the request/response pubsub topics and subscription to use in the test"
}
