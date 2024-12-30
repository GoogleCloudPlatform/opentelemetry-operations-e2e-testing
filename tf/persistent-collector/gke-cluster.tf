// This GKE cluster is used across GKE test runs for running pods, because GKE
// cluster creation/destruction take ~3 minutes each
resource "google_container_cluster" "default" {
  # The terraform workspace will be given a random name (test run id) which we
  # can use to get unique resource names.
  name     = local.gke_cluster_name
  location = local.gke_cluster_location

  initial_node_count = 1
  node_config {
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
  }
}
