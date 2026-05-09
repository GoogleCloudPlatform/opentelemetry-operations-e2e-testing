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

locals {
  service_account_email = "projects/${var.project_id}/serviceAccounts/e2e-cloudbuild-runner@${var.project_id}.iam.gserviceaccount.com"
}

module "python" {
  source          = "../modules/repo-ci-triggers"
  project_id      = var.project_id
  repository      = "opentelemetry-operations-python"
  run_on          = ["local", "gce", "gke", "gae", "gae-standard", "cloud-run", "cloud-functions-gen2"]
  service_account = local.service_account_email
}

module "java" {
  source          = "../modules/repo-ci-triggers"
  project_id      = var.project_id
  repository      = "opentelemetry-operations-java"
  run_on          = ["local", "gce", "gke", "gae", "cloud-run", "cloud-functions-gen2"]
  service_account = local.service_account_email
}

module "js" {
  source          = "../modules/repo-ci-triggers"
  project_id      = var.project_id
  repository      = "opentelemetry-operations-js"
  run_on          = ["local", "gce", "gke", "gae", "gae-standard", "cloud-run", "cloud-functions-gen2"]
  service_account = local.service_account_email
}

module "go" {
  source          = "../modules/repo-ci-triggers"
  project_id      = var.project_id
  repository      = "opentelemetry-operations-go"
  run_on          = ["local", "gce", "gke", "gae", "gae-standard", "cloud-run", "cloud-functions-gen2"]
  service_account = local.service_account_email
}
module "e2e_testing" {
  source          = "../modules/repo-ci-triggers"
  project_id      = var.project_id
  repository      = "opentelemetry-operations-e2e-testing"
  run_on          = ["gke", "gce", "gae", "gae-standard", "cloud-run", "cloud-functions-gen2"]
  service_account = local.service_account_email
}

resource "google_pubsub_topic" "e2e_cleanup" {
  name    = "e2e-cleanup"
  project = var.project_id
}

resource "google_cloudbuild_trigger" "global_cleanup" {
  name            = "global-e2e-cleanup"
  description     = "Global cleanup for E2E tests triggered by Pub/Sub"
  service_account = local.service_account_email

  pubsub_config {
    topic = google_pubsub_topic.e2e_cleanup.id
  }

  filter = "_BUILD_STATUS == 'SUCCESS' || _BUILD_STATUS == 'FAILURE'"

  build {
    step {
      name = "$_TEST_RUNNER_IMAGE"
      args = [
        "cleanup",
        "--project-id=${var.project_id}",
        "--test-run-id=$_TEST_RUN_ID"
      ]
      dir = "/"
    }
    options {
      logging = "CLOUD_LOGGING_ONLY"
    }
  }

  substitutions = {
    _TEST_RUN_ID       = "$(body.message.data.id)"
    _TEST_RUNNER_IMAGE = "$(body.message.data.substitutions._TEST_RUNNER_IMAGE)"
    _BUILD_STATUS      = "$(body.message.data.status)"
  }
}
