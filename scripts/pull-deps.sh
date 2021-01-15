#!/bin/bash

set -e

# Required versions
HELM_VERSION="3.5.0"
HELM_SHA256_SUM="3fff0354d5fba4c73ebd5db59a59db72f8a5bbe1117a0b355b0c2983e98db95b"

IAM_AUTHENTICATOR_VERSION="0.5.2"
IAM_AUTHENTICATOR_SHA256_SUM="5bbe44ad7f6dd87a02e0b463a2aed9611836eb2f40d7fbe8c517460a4385621b"

KUBEVAL_VERSION="0.15.0"
KUBEVAL_SHA256_SUM="70bff2642a2886c0d9ebea452ffb81f333a956e26bbe0826fd7c6797e343e5aa"

KUBECTL_VERSION="v1.20.2"
KUBECTL_SHA512_SUM="e4513cdd65ed980d493259cc7eaa63c415f97516db2ea45fa8c743a6e413a0cdaf299d03dd799286cf322182bf9694204884bb0dd0037cf44592ddfa5e51f183"

KIND_VERSION="v0.8.1"
KIND_SHA_256_SUM="781c3db479b805d161b7c2c7a31896d1a504b583ebfcce8fcd49538c684d96bc"

GOOS=linux
GOARCH=amd64

echo "Downloading helm at version ${HELM_VERSION}"
wget -q https://get.helm.sh/helm-v${HELM_VERSION}-${GOOS}-${GOARCH}.tar.gz
echo "${HELM_SHA256_SUM} helm-v${HELM_VERSION}-${GOOS}-${GOARCH}.tar.gz" | sha256sum -c
tar -xzf helm-v${HELM_VERSION}-${GOOS}-${GOARCH}.tar.gz
cp ${GOOS}-${GOARCH}/helm .

echo "Downloading aws-iam-authenticator at version ${IAM_AUTHENTICATOR_VERSION}"
wget -q -O aws-iam-authenticator https://github.com/kubernetes-sigs/aws-iam-authenticator/releases/download/v${IAM_AUTHENTICATOR_VERSION}/aws-iam-authenticator_${IAM_AUTHENTICATOR_VERSION}_${GOOS}_${GOARCH}
echo "${IAM_AUTHENTICATOR_SHA256_SUM} aws-iam-authenticator" | sha256sum -c
chmod +x aws-iam-authenticator

echo "Downloading kubeval at version ${KUBEVAL_VERSION}"
wget -q https://github.com/instrumenta/kubeval/releases/download/${KUBEVAL_VERSION}/kubeval-${GOOS}-${GOARCH}.tar.gz
echo "${KUBEVAL_SHA256_SUM} kubeval-${GOOS}-${GOARCH}.tar.gz" | sha256sum -c
tar -xzf kubeval-${GOOS}-${GOARCH}.tar.gz

echo "Downloading kubectl at version ${KUBECTL_VERSION}"
wget -q https://dl.k8s.io/${KUBECTL_VERSION}/kubernetes-client-${GOOS}-${GOARCH}.tar.gz
echo "${KUBECTL_SHA512_SUM} kubernetes-client-${GOOS}-${GOARCH}.tar.gz" | sha512sum -c
tar -xvzf kubernetes-client-${GOOS}-${GOARCH}.tar.gz
cp kubernetes/client/bin/kubectl .

echo "Downloading kind at version ${KIND_VERSION}"
wget -q -O kind https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-${GOOS}-${GOARCH}
echo "${KIND_SHA_256_SUM} kind" | sha256sum -c
chmod +x kind
