### 🔬 Kubeapply diff result (stage)

#### Cluster: `test-env:test-region:test-cluster1`<br/><br/>Subpaths (1): `test/subpath`


##### Resource `test1`
<details>
<p>
<summary><b>Diffs (2 lines changed)</b></summary>
```diff
line1
line2
line3
```
</p>
</details>
##### Resource `test2`
<details>
<p>
<summary><b>Diffs (2 lines changed)</b></summary>
```diff
line1
line2
```
</p>
</details>
##### Resource `test3`
<details>
<p>
<summary><b>Diffs (10 lines changed)</b></summary>
```diff
line1
line2
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


```
No diffs were found.
```

#### Next steps

- 🤖 To apply these diffs in the cluster, post:
    - `kubeapply apply test-env:test-region:test-cluster2`
- 🌎 To see the status of all current workloads in the cluster, post:
    - `kubeapply status test-env:test-region:test-cluster2`
- 🔬 To re-generate these diffs, post:
    - `kubeapply diff test-env:test-region:test-cluster2`