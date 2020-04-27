# kubectl apply -f https://github.com/nats-io/nats-operator/releases/download/v0.5.0/00-prereqs.yaml
# kubectl apply -f https://github.com/nats-io/nats-operator/releases/download/v0.5.0/10-deployment.yaml

# # Install NATS Streaming Operator on default namespace
# kubectl apply -f https://raw.githubusercontent.com/nats-io/nats-streaming-operator/master/deploy/default-rbac.yaml

# kubectl apply -f https://raw.githubusercontent.com/nats-io/nats-streaming-operator/master/deploy/deployment.yaml

# sleep 10

kubectl apply -f deployment.yaml
