# Copyright 2026 Google LLC
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

resource "google_compute_instance_template" "default" {
  # The terraform workspace will be given a random name (test run id) which we
  # can use to get unique resource names.
  name         = "e2etest-mig-${terraform.workspace}"
  machine_type = "e2-micro"

  labels = local.common_labels

  disk {
    source_image = "projects/cos-cloud/global/images/family/cos-stable"
    auto_delete  = true
    boot         = true
  }

  metadata = {
    google-logging-enabled = "true"
    startup-script = templatefile("./startup_script.sh", {
      image                     = var.image
      registries                = join(",", distinct([split("/", var.image)[0], "us-docker.pkg.dev", "us-central1-docker.pkg.dev", "gcr.io"]))
      project_id                = var.project_id
      request_subscription_name = module.pubsub.info.request_topic.subscription_name
      response_topic_name       = module.pubsub.info.response_topic.topic_name
      subscription_mode         = "pull"
    })
  }

  network_interface {
    network = "default"

    // TODO: remove this to not allocate external IP. Tried but GCE container
    // doesn't seem to boot without public IP
    access_config {
      // Ephemeral IP
    }
  }

  service_account {
    scopes = ["cloud-platform"]
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "google_compute_instance_group_manager" "default" {
  count              = var.regional ? 0 : 1
  name               = "e2etest-mig-${terraform.workspace}"
  base_instance_name = "e2etest-mig-${terraform.workspace}"
  target_size        = 1

  version {
    instance_template = google_compute_instance_template.default.id
  }
}

resource "google_compute_region_instance_group_manager" "default" {
  count              = var.regional ? 1 : 0
  name               = "e2etest-mig-${terraform.workspace}"
  base_instance_name = "e2etest-mig-${terraform.workspace}"
  target_size        = 1

  version {
    instance_template = google_compute_instance_template.default.id
  }
}

module "pubsub" {
  source = "../modules/pubsub"

  project_id = var.project_id
}

variable "image" {
  type = string
}

variable "regional" {
  type    = bool
  default = false
}

output "pubsub_info" {
  value       = module.pubsub.info
  description = "Info about the request/response pubsub topics and subscription to use in the test"
}
