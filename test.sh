#!/bin/bash
set -uo pipefail
: "$1"
export TERM=xterm-256color

go build -gcflags="all=-N -l" && ./betula "${1}" ; test ; stty sane
