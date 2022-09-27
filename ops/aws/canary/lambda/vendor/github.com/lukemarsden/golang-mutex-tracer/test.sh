#!/usr/bin/env bash
set -e
go fmt .
go vet -v .
go test -v --race .
go test -run=XXX -bench=.