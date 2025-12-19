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

data "google_container_cluster" "default" {
  name     = local.gke_cluster_name
  location = local.gke_cluster_location
}

data "google_client_config" "default" {}

provider "kubernetes" {
  host  = "https://${data.google_container_cluster.default.endpoint}"
  cluster_ca_certificate = base64decode(data.google_container_cluster.default.master_auth.0.cluster_ca_certificate)
  token = data.google_client_config.default.access_token
}

resource "kubernetes_namespace_v1" "namespace" {
  metadata {
    name = "operator-${terraform.workspace}"
  }
}

resource "kubernetes_manifest" "collector" {
  manifest = {
    apiVersion = "opentelemetry.io/v1beta1"
    kind       = "OpenTelemetryCollector"

    metadata = {
      name      = "collector"
      namespace = kubernetes_namespace_v1.namespace.metadata[0].name
    }

    spec = {
      image           = var.image
      config          = module.otel_config.config
      mode            = "deployment"
      configVersions  = 3
      ipFamilyPolicy  = "SingleStack"
      managementState = "managed"
      upgradeStrategy = "automatic"
      env = [
        {
          name = "POD_NAME"
          valueFrom = {
            fieldRef = {
              fieldPath = "metadata.name"
            }
          }
        },
        {
          name = "NAMESPACE_NAME"
          valueFrom = {
            fieldRef = {
              fieldPath = "metadata.namespace"
            }
          }
        },
        {
          name  = "CONTAINER_NAME"
          value = "collector"
        },
        {
          name  = "OTEL_RESOURCE_ATTRIBUTES"
          value = "k8s.pod.name=$(POD_NAME),k8s.namespace.name=$(NAMESPACE_NAME),k8s.container.name=$(CONTAINER_NAME)"
        },
      ]
    }
  }
  computed_fields = [
    "metadata.annotations",
    "metadata.labels",
    "metadata.finalizers",
    "spec.targetAllocator",
  ]
}

module "otel_config" {
  source      = "../modules/otel-config"
  test_run_id = terraform.workspace
}

variable "image" {
  type = string
}
