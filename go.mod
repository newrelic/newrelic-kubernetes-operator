module github.com/newrelic/newrelic-kubernetes-operator

go 1.13

require (
	github.com/go-logr/logr v0.1.0
	github.com/maxbrunsfeld/counterfeiter/v6 v6.2.2 // indirect
	github.com/newrelic/newrelic-client-go v0.8.0
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	k8s.io/api v0.0.0-20190918155943-95b840bb6a1f
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90
	sigs.k8s.io/controller-runtime v0.4.0
)
