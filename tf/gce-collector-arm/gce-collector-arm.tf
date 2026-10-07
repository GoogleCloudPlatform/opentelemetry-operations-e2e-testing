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
  machine_type              = "c4a-standard-1"
  zone                      = var.zone
  allow_stopping_for_update = true

  boot_disk {
    initialize_params {
      image = "projects/cos-cloud/global/images/family/cos-arm64-stable"
      architecture = "ARM64"
    }
  }

  metadata = {
    google-logging-enabled    = "true"
    startup-script = templatefile("./startup_script.sh", {
      image  = var.image
      config = jsonencode(module.otel_config.config)
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
}

module "otel_config" {
  source      = "../modules/otel-config"
  test_run_id = terraform.workspace
}

variable "image" {
  type = string
}

variable "zone" {
  description = "Zone for the VM. The test runner tries other zones when this one has no capacity."
  type        = string
  default     = "us-central1-a"
}


