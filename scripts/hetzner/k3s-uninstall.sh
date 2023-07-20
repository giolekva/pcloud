#!/bin/sh

ssh root@135.181.48.180 "k3s-uninstall.sh"
ssh root@65.108.39.172 "k3s-uninstall.sh"
ssh root@65.108.39.171 "k3s-uninstall.sh"

ssh root@135.181.48.180 "sudo rm -rf /root-storage/*"
ssh root@65.108.39.172 "sudo rm -rf /root-storage/*"
ssh root@65.108.39.171 "sudo rm -rf /root-storage/*"
