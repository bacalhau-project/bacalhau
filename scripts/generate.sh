#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# generate.sh
#
# Generate the files for the project.
#
# Usage:
#   generate.sh

go get -u golang.org/x/tools/cmd/stringer
go install golang.org/x/tools/cmd/stringer

go get -u go.uber.org/mock/gomock
go install go.uber.org/mock/gomock

go generate ./...
