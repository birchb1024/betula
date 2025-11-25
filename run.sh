#!/bin/bash
go build -gcflags="all=-N -l" && ./betula $* ; stty sane ; tset
