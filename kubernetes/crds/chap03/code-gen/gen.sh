#!/bin/bash -e

CURRENT_DIR=$(pwd)
REPO_DIR=${CURRENT_DIR}

PROJECT_MODULE="github.com/liuliqiang/admin"
IMAGE_NAME="kubernetes-codegen:latest"

CUSTOM_RESOURCE_NAME="admin"
CUSTOM_RESOURCE_VERSION="v1"

echo "Building codegen Docker image..."
docker build -f "${CURRENT_DIR}/Dockerfile" \
             -t "${IMAGE_NAME}" \
             "${REPO_DIR}"

cmd="./generate-groups.sh all \ll
    "$PROJECT_MODULE/pkg/client" \
    "$PROJECT_MODULE/pkg/apis" \
    $CUSTOM_RESOURCE_NAME:$CUSTOM_RESOURCE_VERSION"

echo "Generating client codes..."
docker run --rm \
           -v "${REPO_DIR}:/go/src/${PROJECT_MODULE}" \
           "${IMAGE_NAME}" $cmd

sudo chown $USER:$USER -R $REPO_DIR/pkg