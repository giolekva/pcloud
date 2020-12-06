ROOT="$(dirname -- $(pwd))"

minikube start --driver=docker

# Traefik
helm repo add traefik https://containous.github.io/traefik-helm-chart
helm repo update
kubectl create namespace traefik
helm --namespace=traefik install traefik traefik/traefik \
     --set additionalArguments="{--providers.kubernetesingress,--global.checknewversion=true}" \
     --set ports.traefik.expose=True

eval $(minikube docker-env)

# Knowledge Graph
cd "$ROOT/controller"
docker build --tag=pcloud-api-server .
kubectl create namespace pcloud
helm --namespace=pcloud install init chart \
     --set image.name=pcloud-api-server \
     --set image.pullPolicy=Never

# Application Manager
cd "$ROOT/appmanager"
docker build --tag=pcloud-app-manager .
kubectl create namespace pcloud-app-manager
helm --namespace=pcloud-app-manager install init chart \
     --set image.name=pcloud-app-manager \
     --set image.pullPolicy=Never

# Event Processor
cd "$ROOT/events"
docker build --tag=pcloud-event-processor .
kubectl create namespace pcloud-event-processor
helm --namespace=pcloud-event-processor install init chart \
     --set image.name=pcloud-event-processor \
     --set image.pullPolicy=Never
