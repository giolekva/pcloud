#!/bin/bash

if [[ ! -f /home/pcloud/SSH_LOGGED_IN ]];
then
    sudo shutdown -r
else
    rm /home/pcloud/SSH_LOGGED_IN
fi
