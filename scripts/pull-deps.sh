#!/bin/bash

set -euo pipefail

# Note this is used by both the publish script and the test script.

# Required versions

# From Achille, "If we use a different version than v3.5.4 the content of
# eks-configuration ends up being reformatted, which will trigger a recreation
# of the Zookeeper cluster for Centrifuge's Kafka in production."
#
# If you are reading this because you're on Apple M1 Silicon and can't install
# this version of Helm, see the notes in the eks-configuration README about how
# to work around this problem: https://github.com/segmentio/eks-configuration
HELM_VERSION="3.5.4"
# Checksums are e.g. here: https://github.com/helm/helm/releases/tag/v3.5.4
HELM_SHA256_SUM="a8ddb4e30435b5fd45308ecce5eaad676d64a5de9c89660b56face3fe990b318"

IAM_AUTHENTICATOR_VERSION="0.5.2"
IAM_AUTHENTICATOR_SHA256_SUM="5bbe44ad7f6dd87a02e0b463a2aed9611836eb2f40d7fbe8c517460a4385621b"

KUBECTL_VERSION="v1.21.14"
KUBECTL_SHA512_SUM="52a98cc64abeea4187391cbf0ad5bdd69b6920c2b29b8f9afad194441e642fb8f252e14a91c095ef1e85a23e5bb587916bd319566b6e8d1e03be5505400f44b4"

KIND_VERSION="v0.8.1"
KIND_SHA_256_SUM="781c3db479b805d161b7c2c7a31896d1a504b583ebfcce8fcd49538c684d96bc"

GOOS=linux
GOARCH=amd64

mkdir -p "${HOME}/local/bin"

echo "Downloading helm at version ${HELM_VERSION}"
wget -q https://get.helm.sh/helm-v${HELM_VERSION}-${GOOS}-${GOARCH}.tar.gz
echo "${HELM_SHA256_SUM} helm-v${HELM_VERSION}-${GOOS}-${GOARCH}.tar.gz" | sha256sum -c
tar -xzf helm-v${HELM_VERSION}-${GOOS}-${GOARCH}.tar.gz
${GOOS}-${GOARCH}/helm version
# try /usr/local/bin (for Dockerfile) and fall back
cp ${GOOS}-${GOARCH}/helm "/usr/local/bin" || cp ${GOOS}-${GOARCH}/helm "${HOME}/local/bin"

echo "Downloading aws-iam-authenticator at version ${IAM_AUTHENTICATOR_VERSION}"
wget -q -O aws-iam-authenticator https://github.com/kubernetes-sigs/aws-iam-authenticator/releases/download/v${IAM_AUTHENTICATOR_VERSION}/aws-iam-authenticator_${IAM_AUTHENTICATOR_VERSION}_${GOOS}_${GOARCH}
echo "${IAM_AUTHENTICATOR_SHA256_SUM} aws-iam-authenticator" | sha256sum -c
chmod +x aws-iam-authenticator

echo "Downloading kubectl at version ${KUBECTL_VERSION}"
wget -q https://dl.k8s.io/${KUBECTL_VERSION}/kubernetes-client-${GOOS}-${GOARCH}.tar.gz
echo "${KUBECTL_SHA512_SUM} kubernetes-client-${GOOS}-${GOARCH}.tar.gz" | sha512sum -c
tar -xvzf kubernetes-client-${GOOS}-${GOARCH}.tar.gz
cp kubernetes/client/bin/kubectl .

echo "Downloading kind at version ${KIND_VERSION}"
wget -q -O kind https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-${GOOS}-${GOARCH}
echo "${KIND_SHA_256_SUM} kind" | sha256sum -c
chmod +x kind

# https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip -q awscliv2.zip
mkdir -p "${HOME}/local/aws"
./aws/install --install-dir "${HOME}/local/aws" --bin-dir "${HOME}/local/bin"
