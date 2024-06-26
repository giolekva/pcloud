#!/bin/sh

helm repo add dgraph https://charts.dgraph.io
helm --namespace=dgraph install --create-namespace init dgraph/dgraph \
     --set fullnameOverride=dgraph \
     --set image.repository=dgraph/dgraph \
     --set image.tag=v20.03.7 \
     --set ratel.enabled=False \
     --set zero.replicaCount=1 \
     --set zero.persistence.size=1Gi \
     --set zero.persistence.storageClass=local-path \
     --set alpha.replicaCount=1 \
     --set alpha.persistence.size=1Gi \
     --set alpha.persistence.storageClass=local-path \
     --set alpha.configFile."config\.yaml"="whitelist: '0.0.0.0:255.255.255.255'"

echo "Waiting for dgraph-alpha to start"
sleep 2
kubectl -n dgraph wait --timeout=-1s --for=condition=Ready pod/dgraph-alpha-0
echo "Waiting for dgraph-zero to start"
kubectl -n dgraph wait --timeout=-1s --for=condition=Ready pod/dgraph-zero-0
