#!/bin/sh
packer init groundhog-ami.pkr.hcl
packer build -var-file="variables.pkrvars.hcl" groundhog-ami.pkr.hcl
