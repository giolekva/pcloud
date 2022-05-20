#!/bin/bash

MY_IPS=( $(ifconfig | grep -Eo "([0-9]*\.){3}[0-9]*") )
echo $MY_IPS
SERVERS=(192.168.0.111 192.168.0.112 192.168.0.113 192.168.0.114 192.168.0.116)

sleep $[ ( $RANDOM % 180 )  + 1 ]

for IP in "${SERVERS[@]}"
do
    if [[ ! "${MY_IPS[*]}" =~ "$IP" ]];
    then
	echo $IP
	ssh -oStrictHostKeyChecking=no "pcloud@${IP}" "touch ~/SSH_LOGGED_IN"
    fi
done
