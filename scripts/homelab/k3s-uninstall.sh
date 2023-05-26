#!/bin/sh

USER=pcloud

ssh pcloud@192.168.0.13 "k3s-agent-uninstall.sh"
ssh pcloud@192.168.0.12 "k3s-agent-uninstall.sh"
ssh pcloud@192.168.0.11 "k3s-uninstall.sh"
