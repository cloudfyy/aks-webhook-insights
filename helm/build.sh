#!/bin/bash
export DEBIAN_FRONTEND=noninteractive
APPVER=1.0.0
Version=1.0.0
# run helm package

mkdir ../helm_package
helm lint --values ../helm-lint/values.yaml .
helm package . -d ../helm_package --app-version ${APPVER} --version ${Version}