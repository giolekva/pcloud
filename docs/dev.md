# Development Environment

## Prerequisites
PCloud uses following tools to build and deploy it's packages:
* Python - Used by Bazel: https://www.python.org/downloads/
* Bazel - Build tool of the: https://docs.bazel.build/versions/3.7.0/install.html
* Docker - To build container images: https://docs.docker.com/get-docker/
* k3d - To create local Kubernetes cluster for development environment: https://k3d.io/#installation
* kubectl - To interacto with the cluster: https://kubectl.docs.kubernetes.io/installation/kubectl/
* Helm - To package and distributes both PCloud core services and applications running on it: https://helm.sh/docs/intro/install/

Each of these tools provide multiple ways of installing them, choose the one which best suits you and your Host OS.
To check if requirements are met please run:
```shell
> ./dev/check_requirements.sh
```

# Development Instructions

## Installing PCloud
PCloud installation is two step process:
```shell
> ./dev/create_dev_cluster.sh  # creates local Kubernetes cluster
> ./dev/install_core_services.sh  # installs PCloud core services
```

## Installing applications
Under apps/ directory one can find number of sample application for PCloud. Installing any of those requires building container image, creating Helm Chart tarball and uploading it to Application Manager.
Let's see what that looks like for rpuppy application:
```shell
> bazel run //apps/rpuppy:push_to_dev  # builds and pushes container image to localhost:30500 container registry, which is running inside PCloud cluster
> bazel build //apps/rpuppy:chart  # creates Helm Chart tartball which can be found at bazel-bin/apps/rpuppy/chart.tar.gz
```

## Redeploying core services
To redeploy one of the core services after making changes in it you have to rebuild container images and restart the running service. Let's see what that looks like for API Service:
```shell
> bazel run //core/api:push_to_dev  # builds and pushes container image to localhost:30500 container registry, which is running inside PCloud cluster
> kubectl -n pcloud rollout restart deployment/api
```

Reflecting changes in the chart configuration requires reinstallation of the chart itself:
```shell
> bazel build //core/api:chart  # creates chart tarball
> bazel run //core/api:uninstall  # uninstalls old Helm Chart from the cluster
> bazel run //core/api:install  # installs new Helm Chart to the cluster
```
