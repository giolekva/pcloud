#!/bin/sh

# # # helm repo add cilium https://helm.cilium.io/
# # # helm repo add rook-release https://charts.rook.io/release

# helm repo add bitnami https://charts.bitnami.com/bitnami

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

# k3sup join \
#       --k3s-channel stable \
#       --ip 192.168.0.114 \
#       --user pcloud \
#       --server-user pcloud \
#       --server-ip 192.168.0.111

# k3sup join \
#       --k3s-channel stable \
#       --ip 192.168.0.116 \
#       --user pcloud \
#       --server-user pcloud \
#       --server-ip 192.168.0.111



#source installer/metallb.sh
source installer/ingress-nginx.sh
#source installer/cert-manager.sh
#source installer/longhorn.sh
#source installer/pihole.sh
#source installer/matrix.sh
# source installer/auth.sh

# kubectl apply -f ../../apps/rpuppy/install.yaml

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

# kubectl apply -f ../../apps/maddy/install.yaml
# kubectl apply -f maddy-config.yaml
## maddyctl -config /etc/maddy/config/maddy.conf creds create *****@lekva.me
## maddyctl -config /etc/maddy/config/maddy.conf imap-acct create *****@lekva.me
# kubectl apply -f ../../apps/alps/install.yaml


## kubectl -n ingress-nginx get secret cert-wildcard.lekva.me -o yaml > cert-wildcard.lekva.me.yaml
## kubectl apply -f cert-wildcard.lekva.me.yaml -n app-matrix
