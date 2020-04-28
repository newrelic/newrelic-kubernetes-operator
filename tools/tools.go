// +build tools

package tools

import (
	_ "github.com/maxbrunsfeld/counterfeiter/v6"

	// build/generate.mk
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"

	// build/release.mk
	_ "github.com/goreleaser/goreleaser"
)

// This file imports packages that are used when running go generate, or used
