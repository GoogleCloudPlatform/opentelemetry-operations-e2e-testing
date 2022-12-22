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

resource "google_cloudfunctions2_function" "function" {
  name     = "e2etest-${terraform.workspace}"
  location = "us-central1"

  timeouts {
    create = "5m"
  }

  build_config {
    runtime     = var.runtime
    entry_point = var.entrypoint
    source {
      storage_source {
        bucket = google_storage_bucket.bucket.name
        object = google_storage_bucket_object.object.name
      }
    }
  }

  service_config {
    max_instance_count = 1
    available_memory   = "256M"
    timeout_seconds    = 120
    environment_variables = {
      # Environment variables available during function execution
      "PROJECT_ID"          = var.project_id
      "RESPONSE_TOPIC_NAME" = module.pubsub.info.response_topic.topic_name
    }
    ingress_settings               = "ALLOW_INTERNAL_ONLY"
    all_traffic_on_latest_revision = true
  }

  event_trigger {
    trigger_region = "us-central1"
    event_type     = "google.cloud.pubsub.topic.v1.messagePublished"
    pubsub_topic   = module.pubsub.info.request_topic.topic_id
    retry_policy   = "RETRY_POLICY_RETRY"
  }
}

resource "google_storage_bucket" "bucket" {
  name                        = "${terraform.workspace}-gcf-source" # Every bucket name must be globally unique
  location                    = "US"
  uniform_bucket_level_access = true
}

resource "google_storage_bucket_object" "object" {
  name   = "function-source.zip"
  bucket = google_storage_bucket.bucket.name
  # Add path to the zipped function source code, source file zip should be in the bucket
  source = var.functionsource
}

module "pubsub" {
  source = "../modules/pubsub"

  project_id = var.project_id
}

variable "runtime" {
  type = string
}

variable "entrypoint" {
  type = string
}

variable "functionsource" {
  type = string
}

output "pubsub_info" {
  value       = module.pubsub.info
  description = "Info about the request/response pubsub topics and subscription to use in the test"
}
