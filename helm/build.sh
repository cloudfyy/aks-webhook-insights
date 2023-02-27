#!/bin/bash
export DEBIAN_FRONTEND=noninteractive

# run helm package

mkdir ../helm_package
helm lint 
helm package . -d ../helm_package --app-version 0.1.2 --version 0.1.2 