#!/bin/sh

helm install --create-namespace \
     --namespace longhorn-system \
     longhorn longhorn/longhorn \
     --set defaultSettings.defaultDataPath=/pcloud-storage/longhorn \
     --set persistence.defaultClassReplicaCount=2 \
     --set ingress.enabled=true \
     --set ingress.ingressClassName=nginx-private \
     --set ingress.tls=true \
     --set ingress.host=longhorn.pcloud \
     --set ingress.annotations."cert-manager\.io/cluster-issuer"="selfsigned-ca" \
     --set ingress.annotations."acme\.cert-manager\.io/http01-edit-in-place"="\"true\""
