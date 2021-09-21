#!/bin/bash
set -uo pipefail
: "$1"
go build -gcflags="all=-N -l" && ./betula "${1}" ; test ; stty sane
