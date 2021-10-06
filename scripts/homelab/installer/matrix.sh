#!/bin/sh

kubectl apply -f ../../apps/matrix/install.yaml
export KUBE_EDITOR="emacs -nw" && k edit configmap config -n app-matrix
helm install --create-namespace postgresql bitnami/postgresql \
     --namespace app-matrix \
     --set image.repository=arm64v8/postgres \
     --set image.tag=13.4 \
     --set image.pullPolicy=IfNotPresent \
     --set persistence.size=100Gi \
     --set securityContext.enabled=true \
     --set securityContext.fsGroup=0 \
     --set containerSecurityContext.enabled=true \
     --set containerSecurityContext.runAsUser=0 \
     --set volumePermissions.securityContext.runAsUser=0 \
     --set service.type=ClusterIP \
     --set service.port=5432 \
     --set postgresqlUsername=postgres \
     --set postgresqlPassword=foo \
     --set initdbScripts."createuser\.sh"="echo foo | createuser --pwprompt synapse_user" \
     --set initdbScripts."createdb\.sh"="createdb --encoding=UTF8 --locale=C --template=template0 --owner=synapse_user synapse"

kubectl apply -f www.yaml
