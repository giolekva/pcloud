#!/bin/bash

kubectl delete namespace app-object-store
kubectl delete namespace app-rpuppy
kubectl delete namespace app-minio-importer
kubectl delete namespace app-face-detection
kubectl delete namespace pcloud-app-manager
kubectl apply -f appmanager/install.yaml

sh controller/bootstrap-schema.sh
kubectl -n pcloud rollout restart deployment/api


