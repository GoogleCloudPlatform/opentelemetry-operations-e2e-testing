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


// Resources for requests from test runner -> instrumented test server
resource "google_pubsub_topic" "request" {
  name = "request-${terraform.workspace}"

  labels = merge({
    tf-workspace = terraform.workspace
    },
    var.labels
  )
}

resource "google_pubsub_subscription" "request_subscription" {
  name  = "${google_pubsub_topic.request.name}-pull"
  topic = google_pubsub_topic.request.name

  ack_deadline_seconds = 60

  labels = merge({
    tf-workspace = terraform.workspace
    },
    var.labels
  )

  message_retention_duration = "1200s"

  // TODO: add second subscription with push_config
  // push_config {
  //   push_endpoint = "https://example.com/push"

  //   attributes = {
  //     x-goog-version = "v1"
  //   }
  // }
}

// Resources for responses from instrumented test server -> test runner
resource "google_pubsub_topic" "response" {
  name = "response-${terraform.workspace}"

  labels = merge({
    tf-workspace = terraform.workspace
    },
    var.labels
  )
}

resource "google_pubsub_subscription" "response_subscription" {
  name  = "${google_pubsub_topic.response.name}-pull"
  topic = google_pubsub_topic.response.name

  ack_deadline_seconds = 60

  labels = merge({
    tf-workspace = terraform.workspace
    },
    var.labels
  )

  message_retention_duration = "1200s"
}


variable "project_id" {
  type = string
}

variable "labels" {
  type        = map(string)
  description = "Additional labels to add to the pubsub topic"
  default     = {}
}

output "info" {
  value = {
    request_topic = {
      topic_name        = google_pubsub_topic.request.name
      topic_id          = google_pubsub_topic.request.id
      subscription_name = google_pubsub_subscription.request_subscription.name
    }

    response_topic = {
      topic_name        = google_pubsub_topic.response.name
      subscription_name = google_pubsub_subscription.response_subscription.name
    }
  }
  description = "Info about the request/response pubsub topics and subscription to use in the test"
}
