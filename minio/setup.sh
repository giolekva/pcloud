#!/bin/sh

kubectl create -f operator.yaml
kubectl create namespace minio
kubectl create -n minio -f secrets.yaml
kubectl create -n minio -f deployment.yaml
