### Default Service for Google App Engine (GAE)

This folder contains the Dockerfile that builds the image that runs as the default service for Google App Engine. In order to run e2e tests in GAE environment, they need to be run in individual services. 

GAE requires you to have setup a "default" service before you can create other services. The process of creating and deploying service involves deploying a container with the service code in a container to [GAE flexible](https://cloud.google.com/appengine/docs/flexible) environment.

Use the following steps to build and deploy an image - 
*(You need to have docker installed)*

```
cd gae-service-container 

docker build --tag="us-central1-docker.pkg.dev/opentelemetry-ops-e2e/gae-service-containers/default-service:latest" --file=Dockerfile .

docker push us-central1-docker.pkg.dev/opentelemetry-ops-e2e/gae-service-containers/default-service:latest
```

Running the above commands will upload the code supporting the default service in the Artifact Registry, but does not deploy the service. In order to deploy the service follow the standard procedure of deploying persistent resources. 

The default service needs to be created only once. In case the default service needs to be updated the above commands need to run again **before re-deploying persistent resources** so that the container is updated with the latest changes. 
