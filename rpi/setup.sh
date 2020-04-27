### setup rpi connectivity
## enable wifi and ssh
# host
# set ssid/psswd in wpa_suplicant.conf
cp wpa_supplicant.conf /Volumes/boot/
touch /Volumes/boot/ssh
## attach rpi to ip address
# sudo add rpi to /etc/hosts



### k3s
## create pcloud sudo user
# rpi
sudo adduser pcloud
sudo usermod -aG sudo pcloud
## install k3s without traefik
# pcloud@rpi
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="server --no-deploy traefik" sh
## copy kubeconfig on host
# pcloud@rpi
sudo cp /etc/rancher/k3s/k3s.yaml ~/
sudo chown pcloud k3s.yaml
# host
scp pcloud@rpi:k3s.yaml ~/.k3s.kubeconfig
sed -i -e 's/127\.0\.0\.1/rpi/g' ~/.k3s.kubeconfig
printf "\n\n#k3s kubeconfig\nexport KUBECONFIG=~/.k3s.kubeconfig\n" >> ~/.bash_profile
source ~/.bash_profile
kubectl get pods -A
# pcloud@rpi
rm k3s.yaml
