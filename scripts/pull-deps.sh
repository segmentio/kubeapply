#!/bin/bash

set -e

# Required versions
HELM_VERSION="3.2.4"
HELM_SHA256_SUM="8eb56cbb7d0da6b73cd8884c6607982d0be8087027b8ded01d6b2759a72e34b1"

IAM_AUTHENTICATOR_VERSION="0.5.0"
IAM_AUTHENTICATOR_SHA256_SUM="4ccb4788d60ed76e3e6a161b56e1e7d91da1fcd82c98f935ca74c0c2fa81b7a6"

KUBEVAL_VERSION="0.15.0"
KUBEVAL_SHA256_SUM="70bff2642a2886c0d9ebea452ffb81f333a956e26bbe0826fd7c6797e343e5aa"

KUBECTL_VERSION="v1.17.8"
KUBECTL_SHA512_SUM="87da207547a5fa06836afa7f8fc15af4b8950e4a263367a8eb59eccb2a13bb7f98db2bb5731444fcc5313d80e28d3992579742d8a7ed26c48ae425f16ced449a"

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
