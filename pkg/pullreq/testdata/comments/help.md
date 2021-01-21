### 👋 Kubeapply help (stage)

This repo is configured to use [kubeapply](https://github.com/segmentio/kubeapply).

You can run `kubeapply` commands by posting comments to this pull request:

- `kubeapply help`: Generate this message again based on the latest changes
- `kubeapply diff [optional cluster(s)]`: Generate diffs for either all or the selected cluster(s)
- `kubeapply apply [optional cluster(s)]`: Run `apply` for either all or the selected cluster(s)
- `kubeapply status [optional cluster(s)]`: Show the status of workloads in either all or the selected cluster(s)

Note that "expanding" configs out should be done locally and is not handled by `kubeapply`.


Your change currently affects the following clusters and subpaths:

| Cluster | Subpaths |
| ------- | ------- |
| `test-env:test-region:test-cluster1` | <ul><li>`test/subpath`</li></ul> |
| `test-env:test-region:test-cluster2` | <ul><li>*all*</li></ul> |
| `test-env:test-region:test-cluster3` | <ul><li>`subpath1/subpath2`</li></ul> |
