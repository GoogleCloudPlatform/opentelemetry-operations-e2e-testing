# Copyright 2024 Google LLC
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

# Not enabling the permissions on the default service account here Since the permissions
# bindings specified here will be removed when the terraform cleanup happens, that causes
# failures in app engine environment. These permissions should be granted through the latch-key
# configurations.

resource "google_app_engine_standard_app_version" "test_service" {

  version_id = "v1"
  project    = var.project_id
  service    = "standard-${var.runtime}-${terraform.workspace}"
  runtime    = var.runtime

  deployment {
    zip {
      // The object's media_link does not work, GAE requires this format
      // https://cloud.google.com/appengine/docs/admin-api/reference/rpc/google.appengine.v1#zipinfo
      source_url = "https://storage.googleapis.com/${google_storage_bucket.bucket.name}/${google_storage_bucket_object.object.name}"
    }
  }

  // Required, but if variable is unset (empty string), fall back to the runtime default
  entrypoint {
    shell = var.entrypoint
  }

  env_variables = {
    "PUSH_PORT"                 = "8080",
    "REQUEST_SUBSCRIPTION_NAME" = module.pubsub.info.request_topic.subscription_name,
    "RESPONSE_TOPIC_NAME"       = module.pubsub.info.response_topic.topic_name,
    "PROJECT_ID"                = var.project_id,
    "SUBSCRIPTION_MODE"         = "push"
  }

  noop_on_destroy           = false
  delete_service_on_destroy = true
}

resource "google_storage_bucket" "bucket" {
  name                        = "${terraform.workspace}-appsource" # Every bucket name must be globally unique
  location                    = "US"
  uniform_bucket_level_access = true
}

resource "google_storage_bucket_object" "object" {
  name   = "appsource.zip"
  bucket = google_storage_bucket.bucket.name
  # Add path to the zipped function source code, source file zip should be in the bucket
  source = var.appsource
}

module "pubsub" {
  source = "../modules/pubsub"

  project_id = var.project_id
}

module "pubsub-push-subscription" {
  source = "../modules/pubsub-push-subscription"

  project_id    = var.project_id
  topic         = module.pubsub.info.request_topic.topic_name
  push_endpoint = "https://${google_app_engine_standard_app_version.test_service.service}-dot-${var.project_id}.uc.r.appspot.com/"
}

variable "appsource" {
  type = string
}

variable "runtime" {
  type = string
}

variable "entrypoint" {
  type = string
  default = ""
}

output "pubsub_info" {
  value       = module.pubsub.info
  description = "Info about the request/response pubsub topics and subscription to use in the test"
}
