#!/bin/sh

ROOT=$(pwd)
ROOT=${ROOT%/pcloud*}/pcloud

# Dgraph
source $ROOT/apps/dgraph/install.sh

# Application Manager
bazel run //core/appmanager:push_to_dev
bazel run //core/appmanager:install

# Event Processor
bazel run //core/events:push_to_dev
bazel run //core/events:install

# Knowledge Graph
bazel run //core/api:push_to_dev
source $ROOT/dev/bootstrap_schema.sh
bazel run //core/api:install
