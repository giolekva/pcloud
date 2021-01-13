#!/bin/bash

ROOT=$(pwd)
ROOT=${ROOT%/pcloud*}/pcloud

function run_bazel() {
    if command -v bazel &> /dev/null
    then
	bazel "$@"
    elif command -v bazelisk &> /dev/null
    then
	bazelisk "$@"
    else
	echo "Error: Neither bazel nor bazelisk was found."
	exit 1
    fi
}

# Dgraph
source $ROOT/apps/dgraph/install.sh

# Application Manager
run_bazel run //core/appmanager:push_to_dev
run_bazel run //core/appmanager:install

# Event Processor
run_bazel run //core/events:push_to_dev
run_bazel run //core/events:install

# Knowledge Graph
run_bazel run //core/api:push_to_dev
source $ROOT/dev/bootstrap_schema.sh
run_bazel run //core/api:install
