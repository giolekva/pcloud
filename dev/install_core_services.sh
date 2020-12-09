ROOT="$(dirname -- $(pwd))"

# Knowledge Graph
cd "$ROOT/controller"
docker build --tag=localhost:30500/giolekva/pcloud-api-server .
docker push localhost:30500/giolekva/pcloud-api-server
kubectl create namespace pcloud
helm --namespace=pcloud install init chart \
     --set image.name=localhost:30500/giolekva/pcloud-api-server \
     --set image.pullPolicy=Always

# Application Manager
cd "$ROOT/appmanager"
docker build --tag=localhost:30500/giolekva/pcloud-app-manager .
docker push localhost:30500/giolekva/pcloud-app-manager
kubectl create namespace pcloud-app-manager
helm --namespace=pcloud-app-manager install init chart \
     --set image.name=localhost:30500/giolekva/pcloud-app-manager \
     --set image.pullPolicy=Always

# Event Processor
cd "$ROOT/events"
docker build --tag=localhost:30500/giolekva/pcloud-event-processor .
docker push localhost:30500/giolekva/pcloud-event-processor
kubectl create namespace pcloud-event-processor
helm --namespace=pcloud-event-processor install init chart \
     --set image.name=localhost:30500/giolekva/pcloud-event-processor \
     --set image.pullPolicy=Always
