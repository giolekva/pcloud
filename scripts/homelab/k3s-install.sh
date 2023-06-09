#!/bin/sh

USER=pcloud

K3S_VERSION="v1.27.2+k3s1" # v1.26.3+k3s1"

MASTER_INIT="192.168.0.11"
MASTERS=()
WORKERS=("192.168.0.12" "192.168.0.13" "192.168.0.14" "192.168.0.15")

k3sup install \
      --k3s-channel stable \
      --cluster \
      --user $USER \
      --ip $MASTER_INIT \
      --k3s-version $K3S_VERSION \
      --k3s-extra-args "--disable traefik --disable local-storage --disable servicelb --kube-proxy-arg proxy-mode=ipvs --kube-proxy-arg ipvs-strict-arp --flannel-backend host-gw"

for IP in "${MASTERS[@]}";
do
    k3sup join \
	  --k3s-channel stable \
	  --server \
	  --user $USER \
	  --ip $IP \
	  --server-user $USER \
	  --server-ip $MASTER_INIT \
	  --k3s-version $K3S_VERSION \
	  --k3s-extra-args "--disable traefik --disable local-storage --disable servicelb --kube-proxy-arg proxy-mode=ipvs --kube-proxy-arg ipvs-strict-arp --flannel-backend host-gw"
done


for IP in "${WORKERS[@]}";
do
    k3sup join \
      --k3s-channel stable \
      --ip $IP \
      --user $USER \
      --server-user $USER \
      --server-ip $MASTER_INIT \
      --k3s-version $K3S_VERSION
done
