STATUS: draft

# Bootstrapping PCloud with VPN out of the box
PCloud will come with installation binary automating network setup, installing [Kubernetes](https://kubernetes.io) and running PCloud infrastructure on top of it. This document describes how this process will be automated to bootstrap PCloud and scale it from one server to HA deployment.

## Background
PCloud is a set of tools empowering users to run private deployment of services in a secure by default environment, while maintaining ease of use of Cloud services. To achieve that PCloud will provide fault taulerant runtime environment, utilizing compute and storage resources of  multiple servers. Cummunication channels will be secured by default.

## Goals
* First secure all communication channels.
* For client and server devices provide a way to securely join the infrastructure.
* Provide a way to bootstrap full HA deployment starting from average consumer hardware.

## Non-goals
* PCloud does not provide anynimity, although it can run services doing so.
* Assumes Operating Systems are already installed on all devices. How that can be automated with additional services will be discussed in a separate document.

## Technical overview
**What is to follow has been heavily influenced by [Tailscale](https://tailscale.com), argument can be made that PCloud runs fully on-premise Tailscale deployment.**

pcloud will secure communication channels between all devices using [Wireguard](https://www.wireguard.com) protocol and will provide secure peer-to-peer virtual network. Wireguard uses private/public key cryptography. PCloud will distribute public keys to devices in an acces control list (ACL) respectable way. First we will describe how client devices will gain access to the platform, which will make it easier to undestand how server to server communication will be setup during inital bootstrapping process.

### Client workflow
PCloud will run VPN Coordination service which is accessible from the client device over HTTPS. PCloud client application locally will generate private/public key pair and send public key to the coordination service together with authenticated user token. Private key never leaves client device it was generated on. Coordination service will authenticate/authorize user, record new peer information and respond with IP address to setup Wireguard network interface with, and tripples of (public key, internal IP address, endpoint address)-es describing devices client has access to. Using this information client will setup Wireguard network interface and configure all peers.

### Backend worfklow
Now let's look at how PCloud backend infrsatructure will be boostrapped starting from a single device. Installation binary will contain executables of all infrastructure services. Running it will:
* First generate public/private key pair and use thhem to create Wireguard network interface with 10.0.0.1 IP address.
* Install and run [K3s](https://k3s.io) (Kubernetes distribution) on the machine in the primary mode, attach it to above created network interface and bind to listen only on 10.0.0.1 address.
* Om top K3s deploy all of the PCloud core services among which will be user registration, authentication/authorization and VPN coordination services.
* Submit server (public key, IP address, endpoint address) tripple to the VPN coordinator.
* Prompt client to create first user account and automatically give administrative capabilities to it.

Other servers first will join network exactly same way as client devices do (described above). After which they will go through same installation process as initial server did.

Once infrastructure has enough computing and storage power to sustain High Available (HA) K3s installation without initial server, user can drain and remove K3s node running on initial server with option to degrade server's role in the network from primary server to client.

## Manual installation simulation
Let's say we have three [Raspberry Pi](https://www.raspberrypi.org)-s with hostnames rpi111, rpi112 and rpi113 with Local Area Networ (LAN) IP addresses 192.158.0.111, 192.168.0.112 and 192.168.0.113 respectively.
Instructions bellow set up Wireguard mesh network and run K3s on top of it:

``` bash
rpi111> wg genkey > private
rpi111> cat private
rpi111> cat private | wg genkey > public
rpi111> cat public

rpi112> wg genkey > private
rpi112> cat private
rpi112> cat private | wg genkey > public
rpi112> cat public

rpi113> wg genkey > private
rpi113> cat private
rpi113> cat private | wg genkey > public
rpi113> cat public

rpi111> sudo ip link add wg0 type wireguard
rpi111> sudo ip addr add 10.0.0.1/24 dev wg0
rpi111> sudo wg set wg0 private-key ./private
rpi111> sudo ip link set wg0 upi
rpi111> ip addr
rpi111> wg

rpi112> sudo ip link add wg0 type wireguard
rpi112> sudo ip addr add 10.0.0.2/24 dev wg0
rpi112> sudo wg set wg0 private-key ./private
rpi112> sudo ip link set wg0 upi
rpi112> ip addr
rpi112> wg

rpi113> sudo ip link add wg0 type wireguard
rpi113> sudo ip addr add 10.0.0.3/24 dev wg0
rpi113> sudo wg set wg0 private-key ./private
rpi113> sudo ip link set wg0 upi
rpi113> ip addr
rpi113> wg


rpi111> wg set wg0 peer <RPI112-PUBLIC-KEY> allowed-ips 10.0.0.2/32 endpoint 192.168.0.112:38588
rpi111> wg set wg0 peer <RPI113-PUBLIC-KEY> allowed-ips 10.0.0.3/32 endpoint 192.168.0.113:4077

rpi112> wg set wg0 peer <RPI111-PUBLIC-KEY> allowed-ips 10.0.0.1/32 endpoint 192.168.0.111:38588
rpi112> wg set wg0 peer <RPI113-PUBLIC-KEY> allowed-ips 10.0.0.3/32 endpoint 192.168.0.113:4077

rpi113> wg set wg0 peer <RPI111-PUBLIC-KEY> allowed-ips 10.0.0.1/32 endpoint 192.168.0.111:38588
rpi113> wg set wg0 peer <RPI112-PUBLIC-KEY> allowed-ips 10.0.0.3/32 endpoint 192.168.0.112:4077

rpi111> curl -sfL https://get.k3s.io | K3S_TOKEN=<SOME-SECRET-HERE> INSTALL_K3S_EXEC="server --no-deploy traefik --flannel-iface=wg0 --bind-address=10.0.0.1" K3S_KUBECONFIG_MODE="644" sh

rpi112> curl -sfL https://get.k3s.io | K3S_URL=https://10.0.0.1:6443 K3S_TOKEN=<SOME-SECRET-HERE> INSTALL_K3S_EXEC="agent --flannel-iface=wg0" K3S_KUBECONFIG_MODE="644" sh

rpi113> curl -sfL https://get.k3s.io | K3S_URL=https://10.0.0.1:6443 K3S_TOKEN=<SOME-SECRET-HERE> INSTALL_K3S_EXEC="agent --flannel-iface=wg0" K3S_KUBECONFIG_MODE="644" sh

> kubernetes get nodes -o wide
> kubernetes get pods -o wide -A
```

## Implementation
We might be able to reuse much of the Tailscale's open source [client implementation](https://github.com/tailscale/tailscale "Tailscale client implementation"). Tailscale provides private VPN infrastructure similar way to what was described above and their client implementation compiles for and works on major hardware architecture and operating system combinations. On top of it they solve NAT traversal side of peer-to-peer networking.

Client software will be split into two pieces:
* Background process to maintain connectivity with the PCloud infrastructure.
* Frontend application be it web based or native to the client environment such as Linux, iOS, Android, Windows, MacOS, (*)BSD.

Tailscale provides open  source implementation of the background process and frontend applications for some of the environments.

## Testing
E2E test will set up PCloud infrastructure using virtual machines or containers on top of single machine and will simulate:
* Bootstrapping process described above.
* Client joining PCloud infrastructure.
