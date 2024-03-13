#!/bin/sh

ssh root@h01 "systemctl restart systemd-networkd"
ssh root@h02 "systemctl restart systemd-networkd"
