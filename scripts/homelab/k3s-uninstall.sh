#!/bin/sh

ssh pcloud@46.49.35.44 -p 14 "k3s-agent-uninstall.sh"
ssh pcloud@46.49.35.44 -p 15 "k3s-agent-uninstall.sh"
ssh pcloud@46.49.35.44 -p 11 "k3s-uninstall.sh"
ssh pcloud@46.49.35.44 -p 12 "k3s-agent-uninstall.sh"
ssh pcloud@46.49.35.44 -p 13 "k3s-agent-uninstall.sh"

ssh pcloud@46.49.35.44 -p 11 "sudo rm -rf /pcloud-storage/*"
ssh pcloud@46.49.35.44 -p 12 "sudo rm -rf /pcloud-storage/*"
ssh pcloud@46.49.35.44 -p 13 "sudo rm -rf /pcloud-storage/*"
ssh pcloud@46.49.35.44 -p 14 "sudo rm -rf /pcloud-storage/*"
ssh pcloud@46.49.35.44 -p 15 "sudo rm -rf /pcloud-storage/*"
