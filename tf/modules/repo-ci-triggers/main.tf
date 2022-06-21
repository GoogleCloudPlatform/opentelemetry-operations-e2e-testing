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


// Build the image
resource "google_cloudbuild_trigger" "build_image" {
  description = "Pull request CI to build docker image of e2e test server for ${var.repository}"
  filename    = "cloudbuild-e2e-image.yaml"
  github {
    # shorten the name
    name  = var.repository
    owner = "GoogleCloudPlatform"
    pull_request {
      branch          = "^${var.main_branch}$"
      comment_control = "COMMENTS_ENABLED_FOR_EXTERNAL_CONTRIBUTORS_ONLY"
    }
  }
  name = "${local.repo_short_name}-e2e-build-image"
  tags = [
    local.repo_short_name,
    "build"
  ]
  include_build_logs = "INCLUDE_BUILD_LOGS_WITH_STATUS"
}

// Run tests
resource "google_cloudbuild_trigger" "ci" {
  for_each = var.run_on

  description = "Pull request CI to test ${var.repository}, running tests on ${each.key}"
  filename    = "cloudbuild-e2e-${each.key}.yaml"
  github {
    name  = var.repository
    owner = "GoogleCloudPlatform"
    pull_request {
      branch          = "^${var.main_branch}$"
      comment_control = "COMMENTS_ENABLED_FOR_EXTERNAL_CONTRIBUTORS_ONLY"
    }
  }
  name = "${local.repo_short_name}-e2e-${each.key}"
  tags = [
    local.repo_short_name,
    each.key,
  ]
  include_build_logs = "INCLUDE_BUILD_LOGS_WITH_STATUS"
}

variable "repository" {
  type        = string
  description = "The repository to create the triggers for e.g. opentelemetry-operations-python"
}

variable "run_on" {
  type        = set(string)
  description = "The GCP resources to run the tests on."
  validation {
    condition     = alltrue([for r in var.run_on : contains(["local", "gke", "gce", "cloud-run"], r)])
    error_message = "Variable run_on must be one of: 'local' | 'gke' | 'gce' | 'cloud-run'."
  }
}

variable "main_branch" {
  type        = string
  description = "The main branch of the repository ('main' or 'master')"
  default     = "main"
}

locals {
  # since the trigger name ends up on github checks, it should be short and readable
  repo_short_name = replace(var.repository, "opentelemetry-operations-", "ops-")
}
