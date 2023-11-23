STATUS: draft

# PCloud Developer Tools
This document describes developer tools currently used to develop PCloud, and how they must be changed. One of the first commercial products built on top of the PCloud will be suite of applications needed for small and medium development teams to build their own products. PCloud must be developed using the same suite of tools, so that we can dogfood the commercial product and find problems early.

## Background
PCloud is the private cloud infrastructure with broad set of possibilities. To increase adoption number of first-party products, solving real problems of the real users, must be developed on top of it. Which later can be comercialized to create self-sufficient company.

## Goals
1. Come up with the set of tools needed by dev teams, PCloud team being the first user.
2. These tools must help users during development and testing phases.

## Non-goals
While PCloud platform can host any kind of cloud services, goal of this document is not to dictate how developers will productionize their products developed using PCloud. That is the valid use case by itself, but must be brainstormed and designed separately.

## Tools
PCloud is set of services mainly written in [Go](https://go.dev), packaged as container images and deployed on [Kubernetes](https://kubernetes.io).

### Currently used
1. [Git](https://git-scm.com) based monorepo
2. [Github](https://github.com) to host the code publicly
2. [Makefile](https://www.gnu.org/software/make/manual/make.html)-s to build binaries
3. [Podman](https://podman.io) to package containers
4. [Docker Hub](https://hub.docker.com) to publish container images
5. [k3s](https://k3s.io) to run K8s (Kubernetes) infrastructure
5. [Helm](https://helm.sh) charts to deploy them on top of K8s

PCloud does not currently have unit and integration tests, and accordingly there is no CI/CD (Continuous Integration and Continuous Deployment) system used.

### Tools that can help
1. [Gerrit](https://www.gerritcodereview.com) and [Gitiles](https://gerrit.googlesource.com/gitiles/) to host code and do code reviews
2. [Bazel](https://bazel.build) as a distributed and shared build system. Bazel is multi-lingual and can:
  1. Build binaries and container images
  2. Run tests
  3. Store and re-use built artifacts on shared storage
3. Ticket/bug management - TBD
4. TODO: CI/CD: consider Fluxcd, Argocd, ...
5. TODO: Email service - maddy?
6. [Matrix](https://matrix.org) and [Element](https://matrix.org/ecosystem/clients/element/) for real-time communication
7. [Vaultwarden](https://github.com/dani-garcia/vaultwarden) as OSS implementation of the [Bitwarden](https://bitwarden.com) server, to store and share company wide credentials
8. [KubeVirt](https://kubevirt.io) to run VMs (Virtual Machines) on top of the PCloud. This will be needed to create on-domand workstations for development or manual testing. And to run workers of different services such as Bazel for example. We will most likely have to build the UI (User Interface) for it.
