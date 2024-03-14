#!/bin/sh

ssh root@h01 "systemctl restart systemd-networkd"
ssh root@h02 "systemctl restart systemd-networkd"

ssh root@h01 "tailscale down"
ssh root@h01 "tailscale up --advertise-routes=192.168.100.0/24"
