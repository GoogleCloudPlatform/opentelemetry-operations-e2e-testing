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

resource "google_cloud_run_v2_service" "default" {
  name                = "e2etest-${terraform.workspace}"
  location            = "us-central1"
  deletion_protection = false

  timeouts {
    create = "6m"
  }

  template {
    scaling {
      max_instance_count = 1
    }
    containers {
      image = var.image
      args = ["--config=env:OTEL_CONFIG"]
      ports {
        container_port = 8888
      }
      resources {
        # If true, garbage-collect CPU when once a request finishes
        cpu_idle = false
      }
      env {
        name  = "PROJECT_ID"
        value = var.project_id
      }
      env {
        name = "OTEL_CONFIG"
        value = jsonencode(module.otel_config.config)
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

