#!/bin/bash

# set -e

# trap "exit" INT TERM ERR
# trap "kill 0" EXIT

kubectl -n dgraph port-forward svc/dgraph-alpha 8080 &
sleep 1
curl -X POST http://localhost:8080/admin/schema -d 'enum EventState { NEW PROCESSING DONE } type Ignore { x: Int }'
