// +build tools

package tools

import (
	_ "github.com/maxbrunsfeld/counterfeiter/v6"

	// build/generate.mk
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)

// This file imports packages that are used when running go generate, or used
