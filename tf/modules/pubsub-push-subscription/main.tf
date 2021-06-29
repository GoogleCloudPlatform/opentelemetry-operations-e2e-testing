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

# This module creates a pub/sub push subscription for the given topic and
# push_endpoint.
#
# It is in a separate module from tf/modules/pubsub to avoid circular
# dependencies. E.g. for Cloud Run the push subscription can't be created until
# we know the Cloud Run service's URL to push to.

resource "google_pubsub_subscription" "request_push_subscription" {
  name  = "${var.topic}-sub-push"
  topic = var.topic

  ack_deadline_seconds = 60

  labels = merge({
    tf-workspace = terraform.workspace
    },
    var.labels
  )

  message_retention_duration = "1200s"

  push_config {
    push_endpoint = var.push_endpoint

    oidc_token {
      service_account_email = "e2e-pubsub-invoker@${var.project_id}.iam.gserviceaccount.com"
    }
  }
}

variable "project_id" {
  type = string
}

variable "topic" {
  type        = string
  description = "The topic to create the subscription for"
}

variable "labels" {
  type        = map(string)
  description = "Additional labels to add to the pubsub topic"
  default     = {}
}

variable "push_endpoint" {
  type        = string
  description = "The http endpoint to use for pubsub push subscription."
}

output "subscription_name" {
  value       = google_pubsub_subscription.request_push_subscription.name
  description = "Info about the request/response pubsub topics and subscription to use in the test"
}
