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
      env {
        name = "POD_NAME"
        value_from {
          field_ref {
            field_path = "metadata.name"
          }
        }
      }
      env {
        name = "NAMESPACE_NAME"
        value_from {
          field_ref {
            field_path = "metadata.namespace"
          }
        }
      }
      env {
        name  = "CONTAINER_NAME"
        value = "fake-container-name"
      }
      env {
        name  = "OTEL_RESOURCE_ATTRIBUTES"
        value = "k8s.pod.name=$(POD_NAME),k8s.namespace.name=$(NAMESPACE_NAME),k8s.container.name=$(CONTAINER_NAME)"
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
  value       = module.pubsub.info
  description = "Info about the request/response pubsub topics and subscription to use in the test"
}
