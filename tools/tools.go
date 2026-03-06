//go:build tools
// +build tools

package tools

import (
	// project-specific
	_ "github.com/maxbrunsfeld/counterfeiter/v6"

	// build/lint.mk
	_ "github.com/client9/misspell/cmd/misspell"
	_ "github.com/llorllale/go-gitlint/cmd/go-gitlint"
	_ "github.com/psampaz/go-mod-outdated"
	_ "golang.org/x/tools/cmd/goimports"

	// build/document.mk
	_ "github.com/git-chglog/git-chglog/cmd/git-chglog"
	_ "golang.org/x/tools/cmd/godoc"

	// build/generate.mk
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"

	// build/test.mk
	_ "github.com/stretchr/testify/assert"
)

// This file imports packages that are used when running go generate, or used
