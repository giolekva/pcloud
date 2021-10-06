#!/bin/sh

helm upgrade --create-namespace \
     --namespace ingress-nginx \
     nginx ingress-nginx/ingress-nginx \
     --version 4.0.3 \
     --set fullNameOverride=nginx \
     --set controller.service.type=LoadBalancer \
     --set controller.ingressClassByName=true \
     --set controller.ingressClassResource.name=nginx \
     --set controller.ingressClassResource.enabled=true \
     --set controller.ingressClassResource.default=true \
     --set controller.ingressClassResource.controllerValue="k8s.io/ingress-nginx" \
     --set controller.extraArgs.default-ssl-certificate=ingress-nginx/cert-wildcard.lekva.me \
     --set controller.config.proxy-body-size="100M" \
     --set tcp.25="app-maddy/maddy:25" \
     --set tcp.143="app-maddy/maddy:143" \
     --set tcp.993="app-maddy/maddy:993" \
     --set tcp.587="app-maddy/maddy:587" \
     --set tcp.465="app-maddy/maddy:465"
# #    --set udp.4242="ingress-nginx-private/lighthouse:4242"

# kubectl create configmap \
# 	-n ingress-nginx-private \
# 	lighthouse-cert \
# 	--from-file ../../apps/nebula/lighthouse-cert/
# kubectl create configmap \
# 	-n ingress-nginx-private \
# 	ca-cert \
# 	--from-file ../../apps/nebula/ca-cert/ca.crt
# kubectl create configmap \
# 	-n ingress-nginx-private \
# 	lighthouse-config \
# 	--from-file ../../apps/nebula/lighthouse.yaml

helm upgrade --create-namespace \
     --namespace ingress-nginx-private \
     nginx ingress-nginx/ingress-nginx \
     --version 4.0.3 \
     --set fullnameOverride=nginx-private \
     --set controller.service.type=ClusterIP \
     --set controller.ingressClassByName=true \
     --set controller.ingressClassResource.name=nginx-private \
     --set controller.ingressClassResource.enabled=true \
     --set controller.ingressClassResource.default=false \
     --set controller.ingressClassResource.controllerValue="k8s.io/ingress-nginx-private" \
     --set controller.extraVolumes[0].name="lighthouse-cert" \
     --set controller.extraVolumes[0].configMap.name="lighthouse-cert" \
     --set controller.extraVolumes[1].name=ca-cert \
     --set controller.extraVolumes[1].configMap.name=ca-cert \
     --set controller.extraVolumes[2].name=config \
     --set controller.extraVolumes[2].configMap.name=lighthouse-config \
     --set controller.extraContainers[0].name=nebula \
     --set controller.extraContainers[0].image=giolekva/nebula:latest \
     --set controller.extraContainers[0].imagePullPolicy=IfNotPresent \
     --set controller.extraContainers[0].securityContext.capabilities.add[0]=NET_ADMIN \
     --set controller.extraContainers[0].securityContext.privileged=true \
     --set controller.extraContainers[0].ports[0].name=nebula \
     --set controller.extraContainers[0].ports[0].containerPort=4242 \
     --set controller.extraContainers[0].ports[0].protocol=UDP \
     --set controller.extraContainers[0].command[0]="nebula" \
     --set controller.extraContainers[0].command[1]="--config=/etc/nebula/config/lighthouse.yaml" \
     --set controller.extraContainers[0].volumeMounts[0].name=lighthouse-cert \
     --set controller.extraContainers[0].volumeMounts[0].mountPath=/etc/nebula/lighthouse \
     --set controller.extraContainers[0].volumeMounts[1].name=ca-cert \
     --set controller.extraContainers[0].volumeMounts[1].mountPath=/etc/nebula/ca \
     --set controller.extraContainers[0].volumeMounts[2].name=config \
     --set controller.extraContainers[0].volumeMounts[2].mountPath=/etc/nebula/config \
     --set controller.config.bind-address="111.0.0.1" \
     --set controller.config.proxy-body-size="0" \
     --set udp.53="pihole/pihole-dns-udp:53" \
     --set tcp.53="pihole/pihole-dns-tcp:53"
