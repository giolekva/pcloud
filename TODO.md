Enable kubernetes internal service ip address subnet
kubectl exec -it headscale-0 -n lekva-app-headscale -- headscale --config=/headscale/config/config.yaml routes enable pcloud-ingress -r 1


pihole disable admin password
pihole -a -p


longhorn storage dir during bootstrap


soft-serve keys in secret for fluxcd bootstrap


create_env should initialize repo with config.yaml

metallb memberlist
kubectl create secret generic -n metallb-system memberlist --from-literal=secretkey="$(openssl rand -base64 128)"

