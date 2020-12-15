#!/bin/sh

ROOT="$(dirname -- $(pwd))"

# Dgraph
source $ROOT/apps/dgraph/install.sh

# Application Manager
bazel run //appmanager:push_to_dev
bazel run //appmanager:install

# Event Processor
bazel run //events:push_to_dev
bazel run //events:install

# Knowledge Graph
bazel run //controller:push_to_dev
source $ROOT/dev/bootstrap_schema.sh
bazel run //controller:install
