#!/bin/sh

ROOT="$(dirname -- $(pwd))"

source $ROOT/apps/dgraph/install.sh

# Knowledge Graph
cd "$ROOT/controller"
bazel run //controller:push_to_dev
kubectl create namespace pcloud
helm --namespace=pcloud install init chart \
     --set image.name=localhost:30500/giolekva/pcloud-api-server \
     --set image.pullPolicy=Always

# Application Manager
cd "$ROOT/appmanager"
bazel run //appmanager:push_to_dev
kubectl create namespace pcloud-app-manager
helm --namespace=pcloud-app-manager install init chart \
     --set image.name=localhost:30500/giolekva/pcloud-app-manager \
     --set image.pullPolicy=Always

# Event Processor
cd "$ROOT/events"
bazel run //events:push_to_dev
kubectl create namespace pcloud-event-processor
helm --namespace=pcloud-event-processor install init chart \
     --set image.name=localhost:30500/giolekva/pcloud-event-processor \
     --set image.pullPolicy=Always
