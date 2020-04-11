#!/bin/sh

export MINIO_ACCESSKEY="gio"
export MINIO_SECRETKEY="p12345678"
export MINIO_HOST="http://localhost:9000"

# -- kubectl apply -n kube-system -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# helm install mio --set accessKey=${MINIO_ACCESSKEY},secretKey=${MINIO_SECRETKEY} stable/minio
read -s
kubectl port-forward svc/mio-minio 9000 &
read -s
mc config host add mio-minio ${MINIO_HOST} ${MINIO_ACCESSKEY} ${MINIO_SECRETKEY}
mc mb mio-minio/input


kubectl apply -n kube-system -f mio-minio-secrets.yaml



helm repo add argo https://argoproj.github.io/argo-helm
helm install my-argo --namespace kube-system argo/argo
read -s
kubectl -n kube-system port-forward deployment/my-argo-server 2746 &
read -s

kubectl apply -n kube-system -f argo-events-crds-install.yaml


kubectl apply -n kube-system -f event-source.yaml
kubectl apply -n kube-system -f gateway.yaml
kubectl apply -n kube-system -f sensor.yaml
