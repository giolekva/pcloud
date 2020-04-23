#!/bin/sh

# kubectl create namespace argo
kubectl apply -n pcloud -f install.yaml

# kubectl apply -n kube-system -f mio-minio-secrets.yaml


# helm repo add argo https://argoproj.github.io/argo-helm
# helm install my-argo --namespace kube-system argo/argo
# read -s
# kubectl -n kube-system port-forward deployment/my-argo-server 2746 &
# read -s

#kubectl apply -n kube-system -f argo-events-crds-install.yaml
#read -s


#kubectl apply -n kube-system -f event-source.yaml
#kubectl apply -n kube-system -f gateway.yaml
#kubectl apply -n kube-system -f sensor.yaml
