#!/bin/bash
set -euo pipefail
set -x

: "$1"
go build -gcflags="all=-N -l" && ./betula $@
tset
stty sane
