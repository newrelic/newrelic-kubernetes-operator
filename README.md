# New Relic Kubernetes Operator

[![Testing](https://github.com/newrelic/newrelic-kubernetes-operator/workflows/Testing/badge.svg)](https://github.com/newrelic/newrelic-kubernetes-operator)
[![Security Scan](https://github.com/newrelic/newrelic-kubernetes-operator/workflows/Security%20Scan/badge.svg)](https://github.com/newrelic/newrelic-kubernetes-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/newrelic/newrelic-cli?style=flat-square)](https://goreportcard.com/report/github.com/newrelic/newrelic-kubernetes-operator)
[![GoDoc](https://godoc.org/github.com/newrelic/newrelic-kubernetes-operator?status.svg)](https://godoc.org/github.com/newrelic/newrelic-kubernetes-operator)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/newrelic/newrelic-kubernetes-operator/blob/master/LICENSE)
[![CLA assistant](https://cla-assistant.io/readme/badge/newrelic/newrelic-kubernetes-operator)](https://cla-assistant.io/newrelic/newrelic-kubernetes-operator)
[![Release](https://img.shields.io/github/release/newrelic/newrelic-kubernetes-operator/all.svg)](https://github.com/newrelic/newrelic-kubernetes-operator/releases/latest)

Operator to manage New Relic resources.

Currently enables management of Alert Policies and NRQL Alert Conditions.

# Quick start test drive from zero, running kubernetes in a docker container locally with kind

Get docker, kubectl, kustomize and kind installed
``` bash
brew cask install docker
brew install kustomize kubernetes-cli kind
```

Create a test cluster with kind

``` bash
kind create cluster --name newrelic
kubectl cluster-info
```

Install cert-manager

``` bash
kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/download/v0.15.0/cert-manager.yaml
```

Note: This takes a minute or two to finish so wait a minute before going on to the next step. 
You can also confirm it's running with the command `kubectl rollout status deployment -n cert-manager cert-manager-webhook`

Install the operator in the test cluster.

``` bash
kustomize build github.com/newrelic/newrelic-kubernetes-operator/config/default/ \
  | kubectl apply -f -
```

# Deploy with a custom container

If you want to deploy the operator in a custom container you can override the image name with a `kustomize` yaml file

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: newrelic-kubernetes-operator-system
resources:
  - github.com/newrelic/newrelic-kubernetes-operator/config/default
images:
  - name: newrelic/k8s-operator:snapshot
    newName: <CUSTOM_IMAGE>
    newTag: <CUSTOM_TAG>>
```

The apply the file with 

``` bash
kustomize build . | kubectl apply -f -
```

# Using the operator

The operator will create and update alert policies and NRQL alert conditions as needed by applying yaml files with `kubectl apply -f <filename>`

Sample yaml file
```
apiVersion: nr.k8s.newrelic.com/v1
kind: Policy
metadata:
  name: my-policy
spec:
  name: k8s created policy
  incident_preference: "PER_POLICY"
  region: "us"
  # API_KEY can be specified directly in the yaml file or via a k8 secret
#  api_key: APIKEY
  api_key_secret:
    name: nr-api-key
    namespace: default
    key_name: api-key
  conditions:
    - spec:
        nrql:
          query: "SELECT count(*) FROM Transactions"
          since_value: "10"
        enabled: true
        terms:
          - threshold: "75.0"
            time_function: "all"
            duration: "5"
            priority: "critical"
            operator: "above"
        name: "K8s generated alert condition"
    - spec:
        nrql:
          query: "SELECT count(*) FROM Transactions"
          since_value: "5"
        enabled: true
        terms:
          - threshold: "150.0"
            time_function: "all"
            duration: "5"
            priority: "critical"
            operator: "above"
        name: "K8s generated alert condition 2"

```

You can also just create NRQL alert conditions directly with files similar to 

```yaml
apiVersion: nr.k8s.newrelic.com/v1
kind: NrqlAlertCondition
metadata:
  name: my-alert-condition
spec:
  nrql:
    query: "SELECT count(*) FROM Transactions"
    since_value: "10"
  enabled: true
  terms:
    - threshold: "75.0"
      time_function: "all"
      duration: "5"
      priority: "critical"
      operator: "above"
  name: "K8s generated alert condition"
  existing_policy_id: 26458
  region: "us"
# API_KEY can be specified directly in the yaml file or via a k8 secret
#  api_key: API_KEY
  api_key_secret:
    name: nr-api-key
    namespace: default
    key_name: api-key

```

Please note the `existing_policy_id` field which must be set to a currently existing policy ID in the account configured

`kubectl describe nrqlalertconditions.nr.k8s.newrelic.com` - describes currently configured alert conditions

`kubectl describe policies.nr.k8s.newrelic.com` - describes currently configured alert conditions


# Uninstall the operator

The Operator can be removed with the reverse of installation, namely building the kubernetes resource files with `kustomize` and running `kubectl delete`

``` bash
kustomize build github.com/newrelic/newrelic-kubernetes-operator/config/default/ | kubectl delete -f -
```


# Development Prerequisites

In addition to the quick start...

Install kubebuilder https://go.kubebuilder.io/quick-start.html to get `etcd` and `kube-apiserver` needed for the tests

Note: The brew kubebuilder package will not provide all the necessary dependencies for running the tests. 

You can run the tests with 
`make test` or directly with ginkgo

`ginkgo --tags integration -r ./`

The lint rules can be run with 
`make lint`

