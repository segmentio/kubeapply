# kubestar

`kubestar` is an experimental tool for converting between YAML and
[skycfg](https://github.com/stripe/skycfg)-based starlark. It's meant to
complement the starlark-related functionality in `kubeapply`.

## Installation

Run the following:

```GO111MODULE="on" go get github.com/segmentio/kubeapply/cmd/kubestar```

The `kubestar` binary will be placed in `$GOPATH/bin`.

## Subcommands

### `star2yaml`

The `star2yaml` subcommand evaluates a starlark entrypoint (i.e., of the form
`def main(ctx):`) and outputs the associated Kubernetes YAML to `stdout`.
Note that conversion is already supported in `kubeapply expand`, so this
subcommand shouldn't be needed if you're already using the former.

#### Usage

```
kubestar star2yaml [star path] [flags]

Flags:
  -h, --help            help for star2yaml
      --vars string     JSON-formatted vars to insert in ctx object

Global Flags:
  -d, --debug   Enable debug logging
```

#### Example

Run the following from the repo root:

```
kubestar star2yaml pkg/star/expand/testdata/app.star \
    --vars '{"key":"value"}'
```

Any `proto: tag has too few fields: "-"` errors are expected and shouldn't
affect the conversion process.

### `yaml2star`

The `yaml2star` subcommand converts one or more YAML-formatted Kubernetes
manifests to a skycfg-compatible starlark file. This file can be either
a top-level entrypoint, i.e. with `def main(ctx):`, or a non-top-level
function that's called from elsewhere.

#### Usage

```
kubestar yaml2star [YAML configs] [flags]

Flags:
      --args stringArray    list of arguments to add to custom (non-main)
                            entrypoint, in key=value format
      --entrypoint string   name of entrypoint (default "main")
  -h, --help                help for yaml2star

Global Flags:
  -d, --debug   Enable debug logging
```

The arg values will be used as the default values in the entrypoint. Also,
if the values are found in the body of the YAML, they'll be substituted
with the variable name.

#### Example

Run the following from the repo root:

```
kubestar yaml2star pkg/star/convert/testdata/deployment.yaml \
    --args 'name=special-config' \
    --entrypoint=my_deployment
```

