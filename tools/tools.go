//go:build tools
// +build tools

// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

package tools

//nolint:all
import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
)
