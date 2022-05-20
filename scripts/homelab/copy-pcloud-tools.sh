#!/bin/bash

SERVERS=(192.168.0.111 192.168.0.112 192.168.0.113 192.168.0.114 192.168.0.116)

# source gather-pub-keys.sh

for IP in "${SERVERS[@]}"
do
    echo $IP
    # ssh "pcloud@${IP}" "rm -rf /home/pcloud/pcloud-tools"
    # ssh "pcloud@${IP}" "mkdir /home/pcloud/pcloud-tools"
    # scp authorized_keys "pcloud@${IP}:/home/pcloud/pcloud-tools"
    # ssh "pcloud@${IP}" "cat /home/pcloud/pcloud-tools/authorized_keys >> /home/pcloud/.ssh/authorized_keys"
    # scp zap-disk.sh "pcloud@${IP}:/home/pcloud/pcloud-tools"
    # scp generate-ssh-key.sh "pcloud@${IP}:/home/pcloud/pcloud-tools"
    scp check-ssh-login.sh "pcloud@${IP}:/home/pcloud/pcloud-tools"
    scp restart-if-no-ssh-login.sh "pcloud@${IP}:/home/pcloud/pcloud-tools"
    scp check-ssh-login-cron "pcloud@${IP}:/home/pcloud/pcloud-tools"
    scp restart-if-no-ssh-login-cron "pcloud@${IP}:/home/pcloud/pcloud-tools"
    ssh "pcloud@${IP}" "sudo mv /home/pcloud/pcloud-tools/check-ssh-login-cron /etc/cron.d/check-ssh-login"
    ssh "pcloud@${IP}" "sudo mv /home/pcloud/pcloud-tools/restart-if-no-ssh-login-cron /etc/cron.d/restart-if-no-ssh-login"
    ssh "pcloud@${IP}" "sudo chown root:root /etc/cron.d/check-ssh-login"
    ssh "pcloud@${IP}" "sudo chown root:root /etc/cron.d/restart-if-no-ssh-login"
done
