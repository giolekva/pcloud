#!/bin/sh

helm upgrade --create-namespace \
     --namespace app-pihole \
     pihole mojo2600/pihole \
     --version 2.4.2 \
     --set image.repository="pihole/pihole" \
     --set image.tag=v5.8.1 \
     --set persistentVolumeClaim.enabled=true \
     --set persistentVolumeClaim.size="5Gi" \
     --set adminPassword="admin" \
     --set ingress.enabled=false \
     --set serviceDhcp.enabled=false \
     --set serviceDns.type=ClusterIP \
     --set serviceWeb.type=ClusterIP \
     --set serviceWeb.http.enabled=true \
     --set serviceWeb.https.enabled=false \
     --set virtualHost="pihole.pcloud" \
     --set resources.requests.cpu="250m" \
     --set resources.limits.cpu="500m" \
     --set resources.requests.memory="100M" \
     --set resources.limits.memory="250M"

     # --set ingress.hosts={"internal.pihole.pcloud"} \
     # --set ingress.tls[0].hosts[0]="internal.pihole.pcloud" \
     # --set ingress.tls[0].secretName="cert-internal.pihole.pcloud" \
     # --set ingress.annotations."kubernetes\.io/ingress\.class"="nginx-private" \
     # --set ingress.annotations."cert-manager\.io/cluster-issuer"="selfsigned-ca" \
     # --set ingress.annotations."acme\.cert-manager\.io/http01-edit-in-place"="\"true\"" \

# specify ingressClassName manually

# kubectl create configmap oauth2-proxy-config -n app-pihole --from-file=installer/pihole-oauth2.cfg
# kubectl apply -f installer/pihole-oauth2-proxy.yaml
