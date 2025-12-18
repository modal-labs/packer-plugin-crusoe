#!/bin/bash

go build

packer plugins install --path /home/ec2-user/packer-plugin-crusoe/packer-plugin-crusoe github.com/modal-labs/crusoe

