#!/bin/bash

USER=root

K3S_VERSION="v1.28.3+k3s2"

MASTER_INIT="192.168.100.1"
MASTERS=("192.168.100.2")
WORKERS=()

# --node-taint dodo=dodo:NoSchedule
k3sup install \
	  --ssh-key ~/.ssh/id_ed25519 \
      --k3s-channel stable \
      --cluster \
      --user $USER \
      --ip $MASTER_INIT \
      --k3s-version $K3S_VERSION \
      --k3s-extra-args "--disable traefik --disable local-storage --disable servicelb --kube-proxy-arg proxy-mode=ipvs --kube-proxy-arg ipvs-strict-arp --flannel-backend wireguard-native"

for IP in "${MASTERS[@]}";
do
    k3sup join \
	  --ssh-key ~/.ssh/id_ed25519 \
	  --k3s-channel stable \
	  --server \
	  --user $USER \
	  --ip $IP \
	  --server-user $USER \
	  --server-ip $MASTER_INIT \
	  --k3s-version $K3S_VERSION \
	  --k3s-extra-args "--disable traefik --disable local-storage --disable servicelb --kube-proxy-arg proxy-mode=ipvs --kube-proxy-arg ipvs-strict-arp --flannel-backend wireguard-native"
done

for IP in "${WORKERS[@]}";
do
    k3sup join \
	  --ssh-key ~/.ssh/id_ed25519 \
      --k3s-channel stable \
      --ip $IP \
      --user $USER \
      --server-user $USER \
      --server-ip $MASTER_INIT \
      --k3s-version $K3S_VERSION
done
