## Persistent testing resources

This module is for creating resources that should be persistent across tests. There are no automated tests for the terraform files in this directory. 

Most of the resources used by these tests are set up and torn down for each test run. There are however some persistent resources which are managed by terraform (under [tf/persistent](../persistent/)) in the default terraform workspace:

- The GKE cluster ([e2etest-default](https://pantheon.corp.google.com/kubernetes/clusters/details/us-central1/e2etest-default/details?project=opentelemetry-ops-e2e)) for running test pods.
- The Cloud Build CI Triggers for each repo ([pantheon link](https://pantheon.corp.google.com/cloud-build/triggers?project=opentelemetry-ops-e2e)).
- The default service for the created GAE application ([pantheon link](https://pantheon.corp.google.com/appengine/services?serviceId=default&versionId=v1&project=opentelemetry-ops-e2e)).

If you want to modify any of the persistent resources, update the terraform configs and [apply](#apply) them, rather than changing them directly in
cloud console UI. If anything goes wrong, [apply](#apply) the terraform configs to get
them back to the desired state.

### Apply

The test runner has a convenience command `apply-persistent` which executes
`terraform apply` on the persistent resources. This command will show you
changes it would make and then ask for your confirmation before executing. It
should be idempotent.

Before you apply the persistent resources, make sure that you have built the docker image so that the changes in the terraform scripts can be reflected - 

```bash
# Build docker image of the test runner with latest changes
docker build . -t opentelemetry-operations-e2e-testing:local
```

You can run the apply-persistent command by copying the following in a shell script - 

```bash
#!/bin/sh
opentelemetry_operations_e2e_testing(){
  docker run \
    -e PROJECT_ID=${PROJECT_ID} \
    -e GOOGLE_APPLICATION_CREDENTIALS=${GOOGLE_APPLICATION_CREDENTIALS} \
    -v "${GOOGLE_APPLICATION_CREDENTIALS}:${GOOGLE_APPLICATION_CREDENTIALS}:ro" \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -it \
    --rm \
    opentelemetry-operations-e2e-testing:local \
    "$@"
}
opentelemetry_operations_e2e_testing apply-persistent
```

and then executing the script using - 

```
sh <your_script_name>
```

The following environment variables should be set before executing the script - 
1. PROJECT_ID - the unique ID of your GCP project.
2. GOOGLE_APPLICATION_CREDENTIALS - path to the JSON file containing Google Credentials, usually it's located in - `${HOME}/.config/gcloud/application_default_credentials.json`.

Example Run would look something like - 

```
TestMain: 2021/06/25 20:42:53 setuptf.go:166: Applying any changes to persistent resources
TestMain: 2021/06/25 20:42:53 setuptf.go:53: Running command: /bin/terraform init -input=false -backend-config=bucket=opentelemetry-ops-e2e-e2e-tfstate
Initializing modules...
- java in ../modules/repo-ci-triggers
- js in ../modules/repo-ci-triggers
- python in ../modules/repo-ci-triggers

Initializing the backend...

Successfully configured the backend "gcs"! Terraform will automatically
use this backend unless the backend configuration changes.

Initializing provider plugins...
- Finding hashicorp/google versions matching "3.67.0"...
- Finding hashicorp/kubernetes versions matching "2.1.0"...
- Installing hashicorp/google v3.67.0...
- Installed hashicorp/google v3.67.0 (signed by HashiCorp)
- Installing hashicorp/kubernetes v2.1.0...
- Installed hashicorp/kubernetes v2.1.0 (signed by HashiCorp)

Terraform has created a lock file .terraform.lock.hcl to record the provider
selections it made above. Include this file in your version control repository
so that Terraform can guarantee to make the same selections by default when
you run "terraform init" in the future.

Terraform has been successfully initialized!

You may now begin working with Terraform. Try running "terraform plan" to see
any changes that are required for your infrastructure. All Terraform commands
should now work.

If you ever set or change modules or backend configuration for Terraform,
rerun this command to reinitialize your working directory. If you forget, other
commands will detect it and remind you to do so if necessary.
TestMain: 2021/06/25 20:42:57 setuptf.go:53: Running command: /bin/terraform workspace select default
TestMain: 2021/06/25 20:42:57 setuptf.go:53: Running command: /bin/terraform apply -input=false -lock-timeout=10m -var=project_id=opentelemetry-ops-e2e
module.python.google_cloudbuild_trigger.build_image: Refreshing state... [id=projects/opentelemetry-ops-e2e/triggers/56a00c43-d2bf-420c-b88a-0dad06af9a2a]
module.java.google_cloudbuild_trigger.build_image: Refreshing state... [id=projects/opentelemetry-ops-e2e/triggers/e18c2fa5-baf9-45a5-b43d-54dea16d0d3a]
module.js.google_cloudbuild_trigger.build_image: Refreshing state... [id=projects/opentelemetry-ops-e2e/triggers/0084aee5-54df-4b14-8312-aa6bf749ab03]
module.java.google_cloudbuild_trigger.ci["gce"]: Refreshing state... [id=projects/opentelemetry-ops-e2e/triggers/394461a8-65e7-49f9-9c11-9f7660f190a2]
module.java.google_cloudbuild_trigger.ci["local"]: Refreshing state... [id=projects/opentelemetry-ops-e2e/triggers/6ddff972-3cd0-4fec-bc59-5767a5aa1657]
module.java.google_cloudbuild_trigger.ci["gke"]: Refreshing state... [id=projects/opentelemetry-ops-e2e/triggers/f4a99a4a-ade4-4b19-8cc9-fe0a0a1aa4fc]
module.python.google_cloudbuild_trigger.ci["gke"]: Refreshing state... [id=projects/opentelemetry-ops-e2e/triggers/089e33e1-ca3d-46a9-af24-2f84e0bd612e]
module.python.google_cloudbuild_trigger.ci["local"]: Refreshing state... [id=projects/opentelemetry-ops-e2e/triggers/02ae7a95-343f-4cfb-a9a5-111cf66e4032]
module.python.google_cloudbuild_trigger.ci["gce"]: Refreshing state... [id=projects/opentelemetry-ops-e2e/triggers/15b62e91-90e2-416f-b4c1-bceb80531d3d]
google_container_cluster.default: Refreshing state... [id=projects/opentelemetry-ops-e2e/locations/us-central1/clusters/e2etest-default]
module.js.google_cloudbuild_trigger.ci["gce"]: Refreshing state... [id=projects/opentelemetry-ops-e2e/triggers/dcf89134-e0e6-428a-bd66-bf3f2cc4f177]
module.js.google_cloudbuild_trigger.ci["local"]: Refreshing state... [id=projects/opentelemetry-ops-e2e/triggers/b2d24f53-06bc-4912-886d-5d99c2ef5e20]
module.js.google_cloudbuild_trigger.ci["gke"]: Refreshing state... [id=projects/opentelemetry-ops-e2e/triggers/e47cd2d2-8c6a-446b-8032-d2ba6879ae86]

No changes. Your infrastructure matches the configuration.

Terraform has compared your real infrastructure against your configuration and found no differences, so no changes are needed.

Apply complete! Resources: 0 added, 0 changed, 0 destroyed.
```

#### Miscellaneous Notes
 - Any IAM related changes like roles and permissions, should not be provided through the terraform scripts here. 