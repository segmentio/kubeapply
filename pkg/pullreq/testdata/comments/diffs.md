### 🔬 Kubeapply diff result (stage)

#### Cluster: `test-env:test-region:test-cluster1`<br/><br/>Subpaths (1): `test/subpath`

<details>
<summary><b>Diffs (2)</b></summary>
<p>


```diff
something
--- file1
+++ file2
+ diff1
- diff2

--- file3
+++ file4
+ diff1
- diff2
```


</p>
</details>

#### Next steps

- 🤖 To apply these diffs in the cluster, post:
    - `kubeapply apply test-env:test-region:test-cluster1`
- 🌎 To see the status of all current workloads in the cluster, post:
    - `kubeapply status test-env:test-region:test-cluster1`
- 🔬 To re-generate these diffs, post:
    - `kubeapply diff test-env:test-region:test-cluster1`

#### Cluster: `test-env:test-region:test-cluster2`<br/><br/>Subpaths (1): *all*

<details>
<summary><b>Diffs (0)</b></summary>
<p>


```
No diffs found.
```


</p>
</details>

#### Next steps

- 🤖 To apply these diffs in the cluster, post:
    - `kubeapply apply test-env:test-region:test-cluster2`
- 🌎 To see the status of all current workloads in the cluster, post:
    - `kubeapply status test-env:test-region:test-cluster2`
- 🔬 To re-generate these diffs, post:
    - `kubeapply diff test-env:test-region:test-cluster2`