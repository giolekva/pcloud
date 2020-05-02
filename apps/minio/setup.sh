#!/bin/sh

kubectl create namespace minio
kubectl -n minio create -f secrets.yaml
helm --namespace minio install minio-initial stable/minio \
     --set fullnameOverride=minio \
     --set image.repository=giolekva/minio-arm \
     --set image.tag=latest \
     --set image.pullPolicy=Always \
     --set existingSecret=minio-creds \
     --set persistence.size=1Gi
kubectl -n minio create -f ingress.yaml


# mc config host add pcloud http://minio:9000 minio minio123
# mc mb pcloud/images
# mc admin config set pcloud notify_webhook:pcloud queue_limit="100" queue_dir="/tmp/events" endpoint="http://minio-importer.app-minio-importer.svc:80/new_object"
# mc admin service restart pcloud
# mc event add pcloud/images arn:minio:sqs::pcloud:webhook --event put
# mc event list pcloud/images
