variable "test_run_id" {
  type = string
}
output "config" {
  value = {
    receivers = {
      "otlp/internal" = {
        protocols = {
          grpc = {
            endpoint = "localhost:14317"
          }
          http = {
            endpoint = "localhost:14318"
          }
        }
      }
    }
    processors = {
      resourcedetection = {
        detectors = ["gcp"]
      }
      transform = {
        error_mode = "ignore"
        metric_statements = [
          {
            context = "datapoint"
            statements = [
              "set(attributes[\"otelcol_google_e2e\"], ${format("%q", var.test_run_id)})"
            ]
          }
        ]
        log_statements = [
          {
            context = "log"
            statements = [
              "set(attributes[\"otelcol_google_e2e\"], ${format("%q", var.test_run_id)})"
            ]
          }
        ]
        trace_statements = [
          {
            context = "spanevent"
            statements = [
              "set(resource.attributes[\"otelcol_google_e2e\"], ${format("%q", var.test_run_id)})"
            ]
          }
        ]
      }
    }
    exporters = {
      googlecloud = {
        log = {
          default_log_name = "google-otelcol/smoke-test"
        }
      }
    }

    extensions = {
      "health_check" = {}
    }

    service = {
      pipelines = {
        metrics = {
          receivers = ["otlp/internal"]
          processors = ["resourcedetection", "transform"]
          exporters = ["googlecloud"]
        }
        traces = {
          receivers = ["otlp/internal"]
          processors = ["resourcedetection", "transform"]
          exporters = ["googlecloud"]
        }
        logs = {
          receivers = ["otlp/internal"]
          processors = ["resourcedetection", "transform"]
          exporters = ["googlecloud"]
        }
      }
      telemetry = {
        logs = {
          processors = [
            {
              batch = {
                exporter = {
                  otlp = {
                    protocol = "http/protobuf"
                    endpoint = "http://localhost:14318"
                  }
                }
              }
            }
          ]
        }
        metrics = {
          address = "0.0.0.0:8888"
          readers = [
            {
              periodic = {
                interval = 10000
                exporter = {
                  otlp = {
                    protocol = "grpc/protobuf"
                    endpoint = "http://localhost:14317"
                  }
                }
              }
            }
          ]
        }
        traces = {
          processors = [
            {
              batch = {
                exporter = {
                  otlp = {
                    protocol = "grpc/protobuf"
                    endpoint = "http://localhost:14317"
                  }
                }
              }
            }
          ]
        }
      }
    }
  }
}
