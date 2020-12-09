#!/bin/bash

ROOT="$(dirname -- $(pwd))"

k3d cluster create pcloud-dev \
    --servers=1 \
    --k3s-server-arg="--disable=traefik" \
    --port="8080:80@loadbalancer"
k3d kubeconfig merge pcloud-dev --switch-context

# Traefik
helm repo add traefik https://containous.github.io/traefik-helm-chart
helm repo update
kubectl create namespace traefik
helm --namespace=traefik install traefik traefik/traefik \
     --set additionalArguments="{--providers.kubernetesingress,--global.checknewversion=true}" \
     --set ports.traefik.expose=True

# Container Registry
kubectl apply -f $ROOT/apps/container-registry/install.yaml
## Right now ingress on container registry does not work for some reason.
## Use kubectl port-forward bellow to expose registry on localhost.
##  kubectl port-forward service/registry -n container-registry 8090:5000
## And add "127.0.0.1 pcloud-dev-container-registry" to /etc/hosts
## After that one can:
##  docker build --tag=pcloud-dev-container-registry:8090/foo/bar:latest .
##  docker push pcloud-dev-container-registry:8090/foo/bar:latest
