#!/bin/bash

if [ -f /home/pcloud/.ssh/id_ed25519 ];
then
    exit 0
fi

ssh-keygen -t ed25519 -b 4096 -N "" -f /home/pcloud/.ssh/id_ed25519
