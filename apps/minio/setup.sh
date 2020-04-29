#!/bin/sh

kubectl create namespace minio
kubectl -n minio create -f secrets.yaml
helm --namespace minio install minio-initial stable/minio \
     --set fullnameOverride=minio \
     --set image.repository=giolekva/minio-arm \
     --set image.tag=latest \
     --set existingSecret=minio-creds
kubectl -n minio create -f ingress.yaml
