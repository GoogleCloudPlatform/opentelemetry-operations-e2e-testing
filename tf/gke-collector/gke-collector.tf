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


resource "kubernetes_pod" "collector" {
  metadata {
    name = "collector-${terraform.workspace}"
  }

  spec {
    container {
      image = var.image
      name  = "collector"
      args = ["--config=env:OTEL_CONFIG"]

      liveness_probe {
        http_get {
          port = "13133"
          path = "/"
        }
      }

      env {
        name  = "PROJECT_ID"
        value = var.project_id
      }
      env {
        name = "OTEL_CONFIG"
        value = jsonencode(module.otel_config.config)
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
        value = "collector"
      }
      env {
        name  = "OTEL_RESOURCE_ATTRIBUTES"
        value = "k8s.pod.name=$(POD_NAME),k8s.namespace.name=$(NAMESPACE_NAME),k8s.container.name=$(CONTAINER_NAME)"
      }
    }
  }
}

module "otel_config" {
  source      = "../modules/otel-config"
  test_run_id = terraform.workspace
}


variable "image" {
  type = string
}
