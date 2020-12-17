#!/bin/sh

function check() {
    name=$1
    installation_instructions=$2
    echo "Checking $name"
    if ! command -v $name &> /dev/null
    then
	echo "  Please make sure you have it installed and can be found in the PATH."
	echo "  Installation instructions can be found at: $installation_instructions"
	return 1
    else
	echo "  Found"
	return 0
    fi
}

missing=0
check "python" "https://www.python.org/downloads/"
missing=$((missing + $?))
check "bazel" "https://docs.bazel.build/versions/3.7.0/install.html"
missing=$((missing + $?))
check "docker" "https://k3d.io/#installation"
missing=$((missing + $?))
check "k3d" "https://k3d.io/#installation"
missing=$((missing + $?))
check "kubectl" "https://kubectl.docs.kubernetes.io/installation/kubectl/"
missing=$((missing + $?))
check "helm" "https://helm.sh/docs/intro/install/"
missing=$((missing + $?))

if (( $missing > 0 ))
then
    echo "Some of the requirements are missing, please see instructions on how to install them above."
    exit 1
else
    echo "All requirements met."
fi
