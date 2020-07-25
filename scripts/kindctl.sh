#!/bin/bash

set -e

REPO_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && cd .. && pwd )"

CLUSTER_NAME=${CLUSTER_NAME:-kubeapply-test}
KIND_KUBECONFIG=${KIND_KUBECONFIG:-${REPO_ROOT}/.kube/kind-${CLUSTER_NAME}.yaml}

check_kind(){
	if ! [ -x "$(command -v kind)" ]; then
		echo "Kind is not installed! Please install and try again."
		exit 1
	fi
}

sub_help(){
	echo "Usage: $ProgName <subcommand> [options]\n"
	echo "Subcommands:"
	echo "    start  Start the kubeapply-test kind cluster"
	echo "    stop   Stop the kubeapply-test kind cluster"
    echo "    reset  Stop then start the kubeapply-test kind cluster"
	echo ""
	echo "For help with each subcommand run:"
	echo "$ProgName <subcommand> -h|--help"
	echo ""
}

sub_start(){
  check_kind
  existing_clusters=$(kind get clusters | grep ${CLUSTER_NAME} || true)

  if [ -n "$existing_clusters" ]; then
    echo "Cluster ${CLUSTER_NAME} already exists!"
    exit 1
  fi

  mkdir -p "${REPO_ROOT}/.kube"

  kind create cluster --name=${CLUSTER_NAME} --kubeconfig=${KIND_KUBECONFIG}
  kubectl cluster-info --context kind-${CLUSTER_NAME} --kubeconfig=${KIND_KUBECONFIG}
}

sub_stop(){
  check_kind

  existing_clusters=$(kind get clusters | grep ${CLUSTER_NAME} || true)

  if [ -z "$existing_clusters" ]; then
    echo "Cluster ${CLUSTER_NAME} is not running"
    exit 1
  fi

  kind delete cluster --name=${CLUSTER_NAME}
  rm -Rf ${KIND_KUBECONFIG}
}

sub_reset(){
  sub_stop
  sub_start
}

subcommand=$1
case $subcommand in
	"" | "-h" | "--help")
		sub_help
		;;
	*)
		shift
		sub_${subcommand} $@
		if [ $? = 127 ]; then
			echo "Error: '$subcommand' is not a known subcommand." >&2
			echo "       Run '$ProgName --help' for a list of known subcommands." >&2
			exit 1
		fi
		;;
esac
