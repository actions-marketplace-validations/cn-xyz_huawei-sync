#!/bin/bash
set -e

export NAMESPACE=${NAMESPACE}
export REMOTE_URL=${REMOTE_URL}
export HUB_LOGIN_URL=${HUB_LOGIN_URL}
export DOCKER_USER=${DOCKER_USER}
export DOCKER_PASS=${DOCKER_PASS}

echo "## Check Package Version ##################"

skopeo --version

echo "## Login dest TRANSPORT ##################"
set -x
skopeo login -u ${DOCKER_USER} -p ${DOCKER_PASS} ${HUB_LOGIN_URL} --tls-verify=false

docker-sync