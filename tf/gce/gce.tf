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

resource "google_compute_instance" "default" {
  # The terraform workspace will be given a random name (test run id) which we
  # can use to get unique resource names.
  name                      = "e2etest-${terraform.workspace}"
  machine_type              = "e2-micro"
  allow_stopping_for_update = true

  labels = merge(
    local.common_labels,
    { container-vm = module.gce_container.vm_container_label },
  )

  boot_disk {
    initialize_params {
      image = module.gce_container.source_image
    }
  }

  metadata = {
    gce-container-declaration = module.gce_container.metadata_value
    google-logging-enabled    = "true"
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
}

module "gce_container" {
  source  = "terraform-google-modules/container-vm/google"
  version = "2.0.0"

  container = {
    image = var.image

    env = [
      {
        name  = "PROJECT_ID"
        value = var.project_id
      },
      {
        name  = "REQUEST_SUBSCRIPTION_NAME"
        value = module.pubsub.info.request_topic.subscription_name
      },
      {
        name  = "RESPONSE_TOPIC_NAME"
        value = module.pubsub.info.response_topic.topic_name
      },
      {
        name  = "SUBSCRIPTION_MODE"
        value = "pull"
      }
    ]
  }
}

module "pubsub" {
  source = "../modules/pubsub"

  project_id = var.project_id
}

variable "image" {
  type = string
}

output "pubsub_info" {
  value       = module.pubsub.info
  description = "Info about the request/response pubsub topics and subscription to use in the test"
}
