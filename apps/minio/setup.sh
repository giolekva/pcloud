#!/bin/sh

kubectl create namespace minio
helm --namespace minio install minio-initial chart/
