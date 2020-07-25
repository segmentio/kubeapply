TEST_KUBECONFIG = .kube/kubeapply-test-cluster.yaml

# Expand out all configs
.PHONY: expand
expand:
	# Clean out current versions of expanded configs
	rm -Rf expanded
	kubeapply expand cluster-local1.yaml

# Validate configs
.PHONY: validate
validate:
	kubeapply validate cluster-local1.yaml

# Diff expanded configs with versions in kube API
.PHONY: diff
diff:
	kubeapply diff cluster-local1.yaml --kubeconfig=$(TEST_KUBECONFIG)

# Apply changes in cluster
.PHONY: apply
apply:
	kubeapply apply cluster-local1.yaml --kubeconfig=$(TEST_KUBECONFIG)

# Start up a local cluster via kind for easier testing
.PHONY: start-kind
start-kind:
	CLUSTER_NAME=kubeapply-test-cluster \
	KIND_KUBECONFIG=$(TEST_KUBECONFIG) \
		./../../scripts/kindctl.sh start
	# We need to manually create the namespace, otherwise kubeapply diff won't work properly.
	kubectl --kubeconfig=$(TEST_KUBECONFIG) create namespace apps

# Stop kind cluster and delete kubeconfig
.PHONY: stop-kind
stop-kind:
	CLUSTER_NAME=kubeapply-test-cluster \
	KIND_KUBECONFIG=$(TEST_KUBECONFIG) \
		./../../scripts/kindctl.sh stop
