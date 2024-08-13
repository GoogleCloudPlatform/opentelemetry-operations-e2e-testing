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

# Create the deployment for the default service in Google App Engine
resource "google_app_engine_flexible_app_version" "default" {
  version_id      = "v1"
  project         = var.project_id
  service         = "default"
  runtime         = "custom"
  service_account = "${var.project_id}@appspot.gserviceaccount.com"

  deployment {
    container {
      image = "us-central1-docker.pkg.dev/${var.project_id}/gae-service-containers/default-service@sha256:70dccd7bdd8e0671fa40283be19fb5a4e4134d0cbaf1cb0d295081c05bdad34b"
    }
  }

  liveness_check {
    path = "/"
  }

  readiness_check {
    path = "/"
  }

  automatic_scaling {
    cool_down_period = "120s"
    cpu_utilization {
      target_utilization = 0.5
    }
  }

  noop_on_destroy = true
}

# Block all external traffic
resource "google_app_engine_service_network_settings" "default" {
  service = google_app_engine_flexible_app_version.default.service
  network_settings {
    ingress_traffic_allowed = "INGRESS_TRAFFIC_ALLOWED_INTERNAL_ONLY"
  }
}
