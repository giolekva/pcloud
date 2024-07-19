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


# # Install runsc
# sudo apt-get update && \
# sudo apt-get install -y \
#     apt-transport-https \
#     ca-certificates \
#     curl \
#     gnupg

# curl -fsSL https://gvisor.dev/archive.key | sudo gpg --dearmor -o /usr/share/keyrings/gvisor-archive-keyring.gpg
# echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/gvisor-archive-keyring.gpg] https://storage.googleapis.com/gvisor/releases release main" | sudo tee /etc/apt/sources.list.d/gvisor.list > /dev/null

# sudo apt-get update && sudo apt-get install -y runsc

# # Install containerd
# # Add Docker's official GPG key:
# sudo apt-get update
# sudo apt-get install ca-certificates curl
# sudo install -m 0755 -d /etc/apt/keyrings
# sudo curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc
# sudo chmod a+r /etc/apt/keyrings/docker.asc

# # Add the repository to Apt sources:
# echo \
#   "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian \
#   $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
#   sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
# sudo apt-get update

# sudo apt-get install containerd.io

# # Configure k3s to use runsc
# copy into /var/lib/rancher/k3s/agent/etc/containerd/config.toml.tmpl

# [plugins.cri.containerd.runtimes.runsc]
#   runtime_type = "io.containerd.runsc.v1"

# systemctl restart k3s

# cat<<EOF | kubectl apply -f -
# apiVersion: node.k8s.io/v1beta1
# kind: RuntimeClass
# metadata:
#   name: gvisor
# handler: runsc
# EOF
