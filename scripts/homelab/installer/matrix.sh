#!/bin/sh

kubectl apply -f ../../apps/matrix/install.yaml
kubectl edit configmap config -n app-matrix
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


## Integrate with PCloud OIDC Provider
## TODO(giolekva): automate secret and config generation
    # oidc_providers:
    #   - idp_id: pcloud
    #     idp_name: "PCloud OIDC Provider"
    #     skip_verification: false
    #     issuer: "https://hydra.lekva.me"
    #     client_id: "matrix"
    #     client_secret: ""
    #     scopes: ["openid", "profile"]
    #     allow_existing_users: true
    #     user_mapping_provider:
    #       config:
    #         localpart_template: "{{ user.username }}"
    #         display_name_template: "{{ user.username }}"
