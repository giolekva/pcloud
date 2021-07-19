#!/bin/sh
mkdir -p __main__/hack
curl -sfL https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-1.14.1-darwin-amd64.tar.gz | tar xvz --strip-components=1 -C __main__/hack
