#!/bin/bash

# Create a bundle for lambda.
#
# Usage:
#   ./scripts/create-lambda-bundle.sh [zip output file name]

set -e

REPO_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && cd .. && pwd )"
DEFAULT_OUTPUT_NAME="lambda.zip"
OUTPUT_NAME=${1:-$DEFAULT_OUTPUT_NAME}
OUTPUT_ZIP=${REPO_ROOT}/${OUTPUT_NAME}

echo "Creating bundle at ${OUTPUT_ZIP}"

pushd ${REPO_ROOT}/build
zip -r9 $OUTPUT_ZIP kubeapply-lambda
popd

TEMP_DIR=$(mktemp -d)

function cleanup {
  rm -rf "${TEMP_DIR}"
  echo "Deleted temp working directory ${TEMP_DIR}"
}

trap cleanup EXIT

pushd ${TEMP_DIR}

$REPO_ROOT/scripts/pull-deps.sh

zip -r9 $OUTPUT_ZIP helm
zip -r9 $OUTPUT_ZIP aws-iam-authenticator
zip -r9 $OUTPUT_ZIP kubeval
zip -r9 $OUTPUT_ZIP kubectl

echo "Created bundle ${OUTPUT_ZIP}"

popd
