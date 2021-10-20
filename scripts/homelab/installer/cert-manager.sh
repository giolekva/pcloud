#!/bin/sh

helm upgrade --create-namespace \
     --namespace cert-manager \
     cert-manager jetstack/cert-manager \
     --version v1.4.0 \
     --set installCRDs=true \
     --set resources.requests.cpu="100m" \
     --set resources.limits.cpu="250m" \
     --set resources.requests.memory="50M" \
     --set resources.limits.memory="150M" \
     --set tolerations[0].key="pcloud" \
     --set tolerations[0].operator="Equal" \
     --set tolerations[0].value="role" \
     --set tolerations[0].effect="NoSchedule" \
     --set cainjector.resources.requests.cpu="100m" \
     --set cainjector.cpu="250m" \
     --set cainjector.resources.requests.memory="50M" \
     --set cainjector.resources.limits.memory="150M" \
     --set cainjector.tolerations[0].key="pcloud" \
     --set cainjector.tolerations[0].operator="Equal" \
     --set cainjector.tolerations[0].value="role" \
     --set cainjector.tolerations[0].effect="NoSchedule" \
     --set webhook.resources.requests.cpu="100m" \
     --set webhook.resources.limits.cpu="250m" \
     --set webhook.resources.requests.memory="50M" \
     --set webhook.resources.limits.memory="150M" \
     --set webhook.tolerations[0].key="pcloud" \
     --set webhook.tolerations[0].operator="Equal" \
     --set webhook.tolerations[0].value="role" \
     --set webhook.tolerations[0].effect="NoSchedule"

kubectl apply -f cert-manager-webhook-gandi/rbac.yaml

helm upgrade --namespace cert-manager  \
     cert-manager-webhook-gandi ./cert-manager-webhook-gandi/deploy/cert-manager-webhook-gandi \
     --set image.repository=giolekva/cert-manager-webhook-gandi \
     --set image.tag=latest \
     --set image.pullPolicy=Always \
     --set logLevel=2 \
     --set resources.requests.cpu="100m" \
     --set resources.limits.cpu="250m" \
     --set resources.requests.memory="50M" \
     --set resources.limits.memory="150M" \
     --set tolerations[0].key="pcloud" \
     --set tolerations[0].operator="Equal" \
     --set tolerations[0].value="role" \
     --set tolerations[0].effect="NoSchedule"

kubectl apply -f cluster-issuer.yaml
kubectl apply -f root-ca-server.yaml
