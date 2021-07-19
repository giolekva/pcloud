if [ ! -f /home/pcloud/SSH_LOGGED_IN ];
then
    echo "SSH_LOGGED_IN not found, restaring"
    sudo shutdown -r
else
    echo "SSH_LOGGED_IN found and removing"
    rm /home/pcloud/SSH_LOGGED_IN
fi
