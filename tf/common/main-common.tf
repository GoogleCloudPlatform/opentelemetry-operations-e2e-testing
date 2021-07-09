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

# This file is symlinked from tf/common/main-common.tf into each terraform
# subdirectory. It has common provider setup and variables.

terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "3.74.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "2.3.2"
    }
  }

  backend "gcs" {
    # Terraform doesn't allow using input variables in backend config. As a
    # workaround, specify bucket when running terraform init with
    # -backend-config="bucket=$PROJECT_ID-e2e-tfstate"

    # bucket  = "${var.project_id}-e2e-tfstate"
    prefix = "terraform/state"
  }
}

provider "google" {
  project = var.project_id
  region  = "us-central1"
  zone    = "us-central1-a"
}

variable "project_id" {
  type = string
}

locals {
  # Common labels to be assigned to all resources
  common_labels = { "otel-e2e-tests" = true }

  # Name of persistent GKE cluster used across tests
  gke_cluster_name     = "e2etest-default"
  gke_cluster_location = "us-central1"
}
