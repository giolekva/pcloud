#!/bin/bash

bash vendor/k8s.io/code-generator/generate-groups.sh \
     all \
     github.com/giolekva/pcloud/core/nebula/generated \
     github.com/giolekva/pcloud/core/nebula/apis \
     "nebula:v1" \
     --go-header-file hack/boilerplate.go.txt
