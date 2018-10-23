#!/usr/bin/env bash

PKGS=(
 k8s.io/kubernetes/pkg/kubectl/cmd
 k8s.io/kubernetes/pkg/kubectl/util/logs
)

version=release-1.12

for pkg in ${PKGS[@]}; do
  echo "go get: $pkg@$version"
  go get -d -insecure $pkg@$version
done

##