resource "google_cloudfunctions2_function" "function" {
  name     = "e2etest-${terraform.workspace}"
  location = "us-central1"

  timeouts {
    create = "1m"
  }

  build_config {
    runtime = var.runtime
    entry_point = var.entrypoint
    source {
      storage_source {
        bucket = google_storage_bucket.bucket.name
        object = google_storage_bucket_object.object.name
      }
    }
  }

  service_config {
    max_instance_count = 3
    min_instance_count = 1
    available_memory = "256M"
    timeout_seconds = 600
    environment_variables = {
      # Environment variables available during function execution
      "PROJECT_ID" = var.project_id
      "REQUEST_SUBSCRIPTION_NAME" = module.pubsub.info.request_topic.subscription_name
      "RESPONSE_TOPIC_NAME" = module.pubsub.info.response_topic.topic_name        
    }
    ingress_settings = "ALLOW_INTERNAL_ONLY"
    all_traffic_on_latest_revision = true
  }

  event_trigger {
    trigger_region = "us-central1"
    event_type = "google.cloud.pubsub.topic.v1.messagePublished"
    pubsub_topic =  module.pubsub.info.request_topic.topic_name
    retry_policy = "RETRY_POLICY_RETRY"
  }
}

resource "google_storage_bucket" "bucket" {
  name                        = "opentelemetry-e2e-test-gcf-source" # Every bucket name must be globally unique
  location                    = "US"
  uniform_bucket_level_access = true
}

resource "google_storage_bucket_object" "object" {
  name   = "function-source.zip"
  bucket = google_storage_bucket.bucket.name
  source = "${var.runtime}/${var.source}" # Add path to the zipped function source code, source file zip should be in the bucket
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
