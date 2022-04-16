#!/bin/sh

USER=pcloud

K3S_VERSION="v1.23.5+k3s1"

MASTER="192.168.0.111"
WORKERS=("192.168.0.112" "192.168.0.113" "192.168.0.114" "192.168.0.116")

k3sup install \
      --k3s-channel stable \
      --cluster \
      --user $USER \
      --ip $MASTER \
      --k3s-version $K3S_VERSION \
      --k3s-extra-args "--node-taint pcloud=role:NoSchedule --disable traefik --disable local-storage --disable servicelb --kube-proxy-arg proxy-mode=ipvs --kube-proxy-arg ipvs-strict-arp --flannel-backend host-gw"

for IP in "${WORKERS[@]}";
do
    k3sup join \
      --k3s-channel stable \
      --ip $IP \
      --user $USER \
      --server-user $USER \
      --server-ip $MASTER \
      --k3s-version $K3S_VERSION
done
