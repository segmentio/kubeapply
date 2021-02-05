#!/bin/bash

diff -u -N $1 $2

# Ensure that we only exit with non-zero status is there was a real error
if [[ $? -gt 1 ]]; then
    exit 1
fi
