#!/bin/sh

ROOT="$(dirname -- $(pwd))"

# Dgraph
source $ROOT/apps/dgraph/install.sh
source $ROOT/dev/bootstrap_schema.sh

# Knowledge Graph
bazel run //controller:push_to_dev
bazel run //controller:install

# Application Manager
bazel run //appmanager:push_to_dev
bazel run //appmanager:install

# Event Processor
bazel run //events:push_to_dev
bazel run //events:install
