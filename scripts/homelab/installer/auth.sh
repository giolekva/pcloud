#!/bin/sh

# helm install --namespace core-auth \
#      postgres \
#      bitnami/postgresql \
#      --set fullnameOverride=postgres \
#      --set image.repository=arm64v8/postgres \
#      --set image.tag=13.4 \
#      --set persistence.size=1Gi \
#      --set securityContext.enabled=true \
#      --set securityContext.fsGroup=0 \
#      --set containerSecurityContext.enabled=true \
#      --set containerSecurityContext.runAsUser=0 \
#      --set volumePermissions.securityContext.runAsUser=0 \
#      --set service.type=ClusterIP \
#      --set service.port=5432 \
#      --set postgresqlPassword=psswd \
#      --set postgresqlDatabase=kratos

# kubectl create configmap kratos -n core-auth --from-file=../../core/auth/kratos.yaml
# kubectl create configmap identity -n core-auth --from-file=../../core/auth/identity.schema.json
# kubectl apply -f ../../core/auth/kratos/install.yaml

kubectl create configmap \
	-n core-auth \
	auth-ui-lighthouse-config \
	--from-file ../../core/auth/ui/lighthouse.yaml

kubectl apply -f ../../core/auth/ui/install.yaml

kubectl create configmap hydra -n core-auth --from-file=../../core/auth/hydra/hydra.yaml
kubectl apply -f ../../core/auth/hydra/install.yaml
