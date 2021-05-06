# Copyright 2021 Google
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

  for_each = setsubtract(
    # existing keys + key for this run
    toset(
      concat(
        data.terraform_remote_state.gcs.outputs.gce_instance_keys,
        [var.test_run_id]
      )
    ),
    # remove the key for this run if we are destroying
    toset(var.destroy_test_run ? [var.test_run_id] : [])
  )

  name         = "debian9-${each.key}"
  machine_type = "e2-micro"

  labels = local.common_labels

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-9"
    }
  }

  network_interface {
    network = "default"

    access_config {
      // Ephemeral IP
    }
  }
}
