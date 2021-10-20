#!/bin/sh

helm upgrade --create-namespace \
     --namespace pihole \
     pihole mojo2600/pihole \
     --version 2.4.2 \
     --set image.repository="pihole/pihole" \
     --set image.tag=v5.8.1 \
     --set persistentVolumeClaim.enabled=true \
     --set persistentVolumeClaim.size="5Gi" \
     --set ingress.enabled=true \
     --set ingress.hosts={"pihole.pcloud"} \
     --set ingress.tls[0].hosts[0]="pihole.pcloud" \
     --set ingress.tls[0].secretName="cert-pihole.pcloud" \
     --set ingress.annotations."kubernetes\.io/ingress\.class"="nginx-private" \
     --set ingress.annotations."cert-manager\.io/cluster-issuer"="selfsigned-ca" \
     --set ingress.annotations."acme\.cert-manager\.io/http01-edit-in-place"="\"true\"" \
     --set serviceDhcp.enabled=false \
     --set serviceDns.type=ClusterIP \
     --set serviceWeb.type=ClusterIP \
     --set serviceWeb.https.enabled=false \
     --set virtualHost="pihole.pcloud" \
     --set resources.requests.cpu="250m" \
     --set resources.limits.cpu="500m" \
     --set resources.requests.memory="100M" \
     --set resources.limits.memory="250M"

# specify ingressClassName manually
