#!/bin/bash

set -euo pipefail

go build

packer plugins install --path $HOME/packer-plugin-crusoe/packer-plugin-crusoe github.com/modal-labs/crusoe

