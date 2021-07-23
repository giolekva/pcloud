#!/bin/sh

# # # helm repo add cilium https://helm.cilium.io/
# # # helm repo add rook-release https://charts.rook.io/release

# helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
# helm repo add jetstack https://charts.jetstack.io
# helm repo add longhorn https://charts.longhorn.io
# helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
# helm repo add mojo2600 https://mojo2600.github.io/pihole-kubernetes/
# # helm repo add kube-state-metrics https://kubernetes.github.io/kube-state-metrics
# # helm repo add grafana https://grafana.github.io/helm-charts
# helm repo update

# ssh -t pcloud@192.168.0.111 "k3s-agent-uninstall.sh"
# ssh -t pcloud@192.168.0.112 "k3s-agent-uninstall.sh"
# ssh -t pcloud@192.168.0.113 "k3s-uninstall.sh"
# ssh -t pcloud@192.168.0.111 "sudo shutdown -r"
# ssh -t pcloud@192.168.0.112 "sudo shutdown -r"
# ssh -t pcloud@192.168.0.113 "sudo shutdown -r"
# ping 192.168.0.113

# k3sup install \
#       --k3s-channel stable \
#       --cluster \
#       --user pcloud \
#       --ip 192.168.0.111 \
#       --k3s-extra-args "--node-taint pcloud=role:NoSchedule --disable traefik --disable local-storage --disable servicelb --kube-proxy-arg proxy-mode=ipvs --kube-proxy-arg ipvs-strict-arp --flannel-backend host-gw"
# #       --k3s-extra-args "--disable-kube-proxy --disable traefik --disable local-storage --disable servicelb --flannel-backend=none"

# k3sup join \
#       --k3s-channel stable \
#       --ip 192.168.0.112 \
#       --user pcloud \
#       --server-user pcloud \
#       --server-ip 192.168.0.111

# k3sup join \
#       --k3s-channel stable \
#       --ip 192.168.0.113 \
#       --user pcloud \
#       --server-user pcloud \
#       --server-ip 192.168.0.111

# kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.10.2/manifests/namespace.yaml
# kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.10.2/manifests/metallb.yaml
# # On first install only
# kubectl create secret generic -n metallb-system memberlist --from-literal=secretkey="$(openssl rand -base64 128)"
# kubectl apply -f metallb-config.yaml



# # # kubectl apply -f bgp-config.yaml
# # helm install cilium cilium/cilium \
# #      --version 1.10.2 \
# #      --namespace kube-system \
# #      --set hubble.relay.enabled=true \
# #      --set hubble.ui.enabled=true \
# #      --set kubeProxyReplacement=strict \
# #      --set k8sServiceHost=192.168.0.113 \
# #      --set k8sServicePort=6443 \
# #      --set policyEnforcementMode=never \
# #      --set nodePort.enabled=true
# #      # --set bgp.enabled=true \
# #      # --set bgp.announce.loadbalancerIP=true \


# # kubectl create ns cilium-test
# # kubectl apply --namespace=cilium-test -f https://raw.githubusercontent.com/cilium/cilium/v1.10.2/examples/kubernetes/connectivity-check/connectivity-check.yaml


# # helm install --create-namespace \
# #      --namespace rook-ceph \
# #      rook-ceph rook-1.6.7/cluster/charts/rook-ceph \
# #      --set image.tag=v1.6.7

# # kubectl apply -f ceph-cluster.yaml
# # # kubectl -n rook-ceph patch cephcluster rook-ceph --type merge -p '{"spec":{"cleanupPolicy":{"confirmation":"yes-really-destroy-data"}}}'
# # # ceph config set mgr mgr/dashboard/server_addr 0.0.0.0


# helm install --create-namespace \
#      --namespace ingress-nginx \
#      nginx ingress-nginx/ingress-nginx \
#      --set fullNameOverride=nginx \
#      --set controller.service.type=LoadBalancer \
#      --set controller.setAsDefaultIngress=true \
#      --set controller.extraArgs.v=2 \
#      --set controller.extraArgs.default-ssl-certificate=ingress-nginx/cert-wildcard.lekva.me


# helm install --create-namespace \
#      --namespace cert-manager \
#      cert-manager jetstack/cert-manager \
#      --version v1.4.0 \
#      --set installCRDs=true

# kubectl apply -f ../../apps/rpuppy/install.yaml


# helm install --create-namespace \
#      --namespace longhorn-system \
#      longhorn longhorn/longhorn \
#      --set defaultSettings.defaultDataPath=/pcloud-storage/longhorn \
#      --set persistence.defaultClassReplicaCount=2 \
#      --set ingress.enabled=true \
#      --set ingress.ingressClassName=nginx \
#      --set ingress.host=longhorn.pcloud \
#      --set ingress.annotations."nginx\.ingress\.kubernetes\.io/ssl-redirect"="\"false\""

# kubectl apply -f ~/dev/src/socialme-go/install.yaml

# # # TODO retention days
# # helm install --create-namespace \
# #      --namespace prometheus \
# #      prometheys prometheus-community/prometheus \  # TODO prometheys
# #      --set alertmanager.ingress.enabled=true \
# #      --set alertmanager.ingress.ingressClassName=nginx \
# #      --set alertmanager.ingress.hosts={alertmanager.prometheus.pcloud} \
# #      --set alertmanager.ingress.annotations."nginx\.ingress\.kubernetes\.io/ssl-redirect"="\"false\"" \
# #      --set server.ingress.enabled=true \
# #      --set server.ingress.ingressClassName=nginx \
# #      --set server.ingress.hosts={prometheus.pcloud} \
# #      --set server.ingress.annotations."nginx\.ingress\.kubernetes\.io/ssl-redirect"="\"false\"" \
# #      --set server.persistentVolume.size=100Gi \
# #      --set pushgateway.ingress.enabled=true \
# #      --set pushgateway.ingress.ingressClassName=nginx \
# #      --set pushgateway.ingress.hosts={pushgateway.prometheus.pcloud} \
# #      --set pushgateway.ingress.annotations."nginx\.ingress\.kubernetes\.io/ssl-redirect"="\"false\"" \
# #      --set pushgateway.persistentVolume.enabled=true

# # helm install --create-namespace \
# #      --namespace grafana \
# #      --set ingress.enabled=true \
# #       --set ingress.ingressClassName=nginx \
# #       --set ingress.hosts={grafana.pcloud} \
# #       --set ingress.annotations."nginx\.ingress\.kubernetes\.io/ssl-redirect"="\"false\"" \
# #       --set persistence.enabled=true \
# #       --set persistence.size=50Gi

# helm install --create-namespace \
#      --namespace prometheus-system \
#      prometheus prometheus-community/kube-prometheus-stack \
#      --set alertmanager.ingress.enabled=true \
#      --set alertmanager.ingress.ingressClassName=nginx \
#      --set alertmanager.ingress.hosts={alertmanager.prometheus.pcloud} \
#      --set alertmanager.ingress.annotations."nginx\.ingress\.kubernetes\.io/ssl-redirect"="\"false\"" \
#      --set alertmanager.ingress.pathType=Prefix \
#      --set grafana.ingress.enabled=true \
#      --set grafana.ingress.ingressClassName=nginx \
#      --set grafana.ingress.hosts={grafana.prometheus.pcloud} \
#      --set grafana.ingress.annotations."nginx\.ingress\.kubernetes\.io/ssl-redirect"="\"false\"" \
#      --set grafana.ingress.pathType=Prefix \
#      --set prometheus.ingress.enabled=true \
#      --set prometheus.ingress.ingressClassName=nginx \
#      --set prometheus.ingress.hosts={prometheus.pcloud} \
#      --set prometheus.ingress.annotations."nginx\.ingress\.kubernetes\.io/ssl-redirect"="\"false\"" \
#      --set prometheus.ingress.pathType=Prefix

# kubectl apply -f ../../apps/pihole/install.yaml
helm upgrade --create-namespace \
     --namespace pihole \
     pihole mojo2600/pihole \
     --set ingress.enabled=true \
     --set ingress.hosts={"pihole.pcloud"} \
     --set serviceDhcp.enabled=false \
     --set serviceDns.type=LoadBalancer \
     --set serviceWeb.type=ClusterIP \
     --set serviceWeb.https.enabled=false \
     --set virtualHost="pihole.pcloud"

# kubectl apply -f cert-manager-webhook-gandi/rbac.yaml
# helm upgrade --namespace cert-manager  \
#      cert-manager-webhook-gandi ./cert-manager-webhook-gandi/deploy/cert-manager-webhook-gandi \
#      --set image.repository=giolekva/cert-manager-webhook-gandi \
#      --set image.tag=latest \
#      --set image.pullPolicy=Always \
#      --set logLevel=2

# kubectl apply -f cluster-issuer.yaml
