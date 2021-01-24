module github.com/newrelic/newrelic-kubernetes-operator

go 1.13

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/go-logr/logr v0.1.0
	github.com/golang/mock v1.4.4
	github.com/goreleaser/goreleaser v0.143.0
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/maxbrunsfeld/counterfeiter/v6 v6.2.3
	github.com/newrelic/go-agent/v3 v3.7.0
	github.com/newrelic/newrelic-client-go v0.39.0
	github.com/onsi/ginkgo v1.13.0
	github.com/onsi/gomega v1.10.1
	github.com/prometheus/client_golang v1.6.0 // indirect
	github.com/prometheus/common v0.10.0 // indirect
	github.com/stretchr/testify v1.6.1
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	golang.org/x/tools v0.0.0-20200724022722-7017fd6b1305
	gomodules.xyz/jsonpatch/v2 v2.1.0 // indirect
	k8s.io/api v0.18.4
	k8s.io/apiextensions-apiserver v0.18.4 // indirect
	k8s.io/apimachinery v0.18.4
	k8s.io/client-go v0.18.4
	k8s.io/utils v0.0.0-20200601170155-a0dff01d8ea5 // indirect
	sigs.k8s.io/controller-runtime v0.6.0
	sigs.k8s.io/controller-tools v0.3.0
)
