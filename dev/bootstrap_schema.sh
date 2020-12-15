#!/bin/bash

kubectl -n dgraph port-forward svc/dgraph-alpha 8081:8080 &
PORT_FORWARD_PID=$!
sleep 1
curl -X POST http://localhost:8081/admin/schema -d 'enum EventState { NEW PROCESSING DONE } type Ignore { x: Int }'
kill -9 $PORT_FORWARD_PID
