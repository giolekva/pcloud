#!/bin/bash

SERVERS=(192.168.0.111 192.168.0.112 192.168.0.113 192.168.0.114 192.168.0.116)

rm -f authorized_keys
touch authorized_keys

for IP in "${SERVERS[@]}"
do
    ssh "pcloud@${IP}" "sh /home/pcloud/pcloud-tools/generate-ssh-key.sh"
    scp "pcloud@${IP}:/home/pcloud/.ssh/id_ed25519.pub" tmp-key.pub
    cat tmp-key.pub >> authorized_keys
    rm tmp-key.pub
done

