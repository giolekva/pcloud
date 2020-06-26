kubectl create namespace pcloud-event-processor
kubectl create namespace pcloud-app-manager
kubectl create namespace pcloud
helm --namespace=pcloud install init ../controller/chart
helm --namespace=pcloud-app-manager install init ../appmanager/chart
helm --namespace=pcloud-event-processor install init ../events/chart
