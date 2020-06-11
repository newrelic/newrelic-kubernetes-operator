// +build tools

package tools

import (
	// project-specific
	_ "github.com/maxbrunsfeld/counterfeiter/v6"

	// build/lint.mk
	_ "golang.org/x/tools/cmd/goimports"

	// build/generate.mk
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"

	// build/release.mk
	_ "github.com/goreleaser/goreleaser"
)

// This file imports packages that are used when running go generate, or used
