#!/usr/bin/env bash

set -euo pipefail

AWS_ACCOUNT_ID='528451384384'
AWS_REGION='us-west-2'

main() {
    local -r SHORT_GIT_SHA=$(git rev-parse --short HEAD)
    local -r image="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/kubeapply:${SHORT_GIT_SHA}"
    local -r lambda_image="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/kubeapply-lambda:${SHORT_GIT_SHA}"
    docker build \
        -t "${image}" \
        --build-arg VERSION_REF="${SHORT_GIT_SHA}" \
        .
    docker push "${image}"

    docker build \
        -f Dockerfile.lambda \
        -t "${lambda_image}" \
        --build-arg VERSION_REF="${SHORT_GIT_SHA}" \
        .
    docker push "${lambda_image}"
}

main "$@"
