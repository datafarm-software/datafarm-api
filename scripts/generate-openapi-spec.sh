#!/usr/bin/env bash
set -e
go run ./cmd/generate-openapi-spec/main.go --filename "api/openapi.generated.yaml"
