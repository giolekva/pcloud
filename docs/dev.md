# Development Environment

## Prerequisites
PCloud uses following tools to build and deploy it's packages:
* Docker - To build container images: https://helm.sh/docs/intro/install/
* k3d - To create local Kubernetes cluster for development environment: https://k3d.io/#installation
* kubectl - To interacto with the cluster: https://kubectl.docs.kubernetes.io/installation/kubectl/
* Helm - To package and distributes both PCloud core services and applications running on it: https://helm.sh/docs/intro/install/

Each of these tools provide multiple ways of installing them, choose the one which best suits you and your Host OS.

## Installing PCloud
PCloud installation is two step process:
1) First local Kubernetes cluster has to be created: ./dev/create_dev_cluster.sh
2) Now we can install PCloud core services top of it: ./dev/install_core_services.sh
