#!/bin/bash

# This is used as the custom differ for kubectl diff. We need a wrapper script instead
# of calling 'kubeapply kdiff' directly because kubectl wants a single executable (without
# any subcommands or arguments).

kubeapply kdiff $1 $2
