### 🔬 Kubeapply diff result (stage)

#### Cluster: `test-env:test-region:test-cluster1`<br/><br/>Subpaths (1): `test/subpath`


#### Resources with diffs (3):
<details>
<summary><b><code>test1</code> (2 lines changed)</b></summary>
<p>

```diff
line1
line2
line3
```

</p>
</details>
<!-- KUBEAPPLY_SPLIT -->

<details>
<summary><b><code>test2</code> (2 lines changed)</b></summary>
<p>

```diff
line1
line2
```

</p>
</details>
<!-- KUBEAPPLY_SPLIT -->

<details>
<summary><b><code>test3</code> (10 lines changed)</b></summary>
<p>

```diff
line1
line2
```

</p>
</details>
<!-- KUBEAPPLY_SPLIT -->


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