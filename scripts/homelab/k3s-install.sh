#!/bin/sh

k3sup install \
      --k3s-channel stable \
      --cluster \
      --user pcloud \
      --ip 192.168.0.111 \
      --k3s-extra-args "--node-taint pcloud=role:NoSchedule --disable traefik --disable local-storage --disable servicelb --kube-proxy-arg proxy-mode=ipvs --kube-proxy-arg ipvs-strict-arp --flannel-backend host-gw"

k3sup join \
      --k3s-channel stable \
      --ip 192.168.0.112 \
      --user pcloud \
      --server-user pcloud \
      --server-ip 192.168.0.111

k3sup join \
      --k3s-channel stable \
      --ip 192.168.0.113 \
      --user pcloud \
      --server-user pcloud \
      --server-ip 192.168.0.111

k3sup join \
      --k3s-channel stable \
      --ip 192.168.0.114 \
      --user pcloud \
      --server-user pcloud \
      --server-ip 192.168.0.111

k3sup join \
      --k3s-channel stable \
      --ip 192.168.0.116 \
      --user pcloud \
      --server-user pcloud \
      --server-ip 192.168.0.111
