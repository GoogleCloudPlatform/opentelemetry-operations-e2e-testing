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

# Enable Logs Writer permission on the service account
resource "google_project_iam_binding" "logs_writer" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  members = [
    "serviceAccount:${var.project_id}@appspot.gserviceaccount.com", 
    ]
}

# Enable Google storage viewer on service account
resource "google_project_iam_binding" "storage_viewer" {
  project = var.project_id
  role    = "roles/storage.objectViewer"
  members = [
    "serviceAccount:${var.project_id}@appspot.gserviceaccount.com", 
    ]
}

# Enable the Artifact Repository pull access 
resource "google_project_iam_binding" "registry_reader" {
  project = var.project_id
  role    = "roles/artifactregistry.reader"
  members = [
    "serviceAccount:${var.project_id}@appspot.gserviceaccount.com", 
    ]
}

# Create the deployment for the default service in Google App Engine
resource "google_app_engine_flexible_app_version" "default" {
  version_id = "v1"
  project    = var.project_id
  service    = "default"
  runtime    = "custom"

  deployment {
    container {
      image = "us-central1-docker.pkg.dev/${var.project_id}/gae-service-containers/default-service:latest"
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
