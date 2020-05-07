# kubectl create namespace dgraph
# helm repo add dgraph https://charts.dgraph.io
# dgraph/dgraph
helm --namespace=dgraph install init /Users/lekva/dev/src/dgraph-charts/charts/dgraph \
     --set fullnameOverride=dgraph \
     --set image.repository=giolekva/dgraph-arm \
     --set image.tag=latest \
     --set zero.replicaCount=1 \
     --set zero.persistence.size=1Gi \
     --set zero.persistence.storageClass=local-path \
     --set alpha.replicaCount=1 \
     --set alpha.persistence.size=1Gi \
     --set alpha.persistence.storageClass=local-path \
     --set ratel.enabled=False \
     --set alpha.whitelist=0.0.0.0:255.255.255.255
