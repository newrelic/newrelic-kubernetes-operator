# New Relic Kubernetes Operator

[![CircleCI](https://circleci.com/gh/newrelic/newrelic-kubernetes-operator.svg?style=svg)](https://circleci.com/gh/newrelic/newrelic-kubernetes-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/newrelic/newrelic-cli?style=flat-square)](https://goreportcard.com/report/github.com/newrelic/newrelic-kubernetes-operator)
[![GoDoc](https://godoc.org/github.com/newrelic/newrelic-kubernetes-operator?status.svg)](https://godoc.org/github.com/newrelic/newrelic-kubernetes-operator)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/newrelic/newrelic-kubernetes-operator/blob/master/LICENSE)
[![CLA assistant](https://cla-assistant.io/readme/badge/newrelic/newrelic-kubernetes-operator)](https://cla-assistant.io/newrelic/newrelic-kubernetes-operator)
[![Release](https://img.shields.io/github/release/newrelic/newrelic-kubernetes-operator/all.svg)](https://github.com/newrelic/newrelic-kubernetes-operator/releases/latest)

Operator to manage New Relic resources.

Currently enables management of Alert Policies and NRQL Alert Conditions.

# Quick start test drive from zero

Get docker, kubectl, and kind installed
``` bash
brew cask install docker
brew install kustomize kubernetes-cli kind
```

Create a test cluster

``` bash
kind create cluster --name newrelic
kubectl cluster-info
```

Install cert-manager

``` bash
kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/download/v0.15.0/cert-manager.yaml
```

Install the operator in the test cluster.

``` bash
kustomize build https://github.com/newrelic/newrelic-kubernetes-operator/config/default \
  | kubectl apply -f -
```

# Deploy with a custom container

``` bash
# TBD
kustomize something something $DOCKER_IMAGE | kubectl apply -
```

# Development Prerequisites

In addition to the quick start...

```bash
brew install kubebuilder
```

# Install the operator in a cluster

*Note: this will install operator on whatever kubernetes cluster `kubectl` is configured to use.*

```bash
$ kubectl api-resources > resources-before.txt

$ make install

$ kubectl api-resources > resources-installed.txt

$ diff -u resources-before.txt resources-installed.txt
```

**Confirm the installation:**

```diff
--- resources-before.txt	2020-01-23 12:55:32.000000000 -0700
+++ resources-installed.txt	2020-01-23 12:55:53.000000000 -0700
@@ -40,6 +40,7 @@
 ingresses                         ing          networking.k8s.io              true         Ingress
 networkpolicies                   netpol       networking.k8s.io              true         NetworkPolicy
 runtimeclasses                                 node.k8s.io                    false        RuntimeClass
+nrqlalertconditions                            nr.k8s.newrelic.com     true         NrqlAlertCondition
 poddisruptionbudgets              pdb          policy                         true         PodDisruptionBudget
 podsecuritypolicies               psp          policy                         false        PodSecurityPolicy
 clusterrolebindings                            rbac.authorization.k8s.io      false        ClusterRoleBinding
```

Now install a Certificate manager, we recommend https://cert-manager.io/docs/installation/kubernetes/#installing-with-regular-manifests

Next, build the image and push it to the desired docker repo

`make docker-build docker-push DOCKER_IMAGE=<some-registry>/<project-name>:tag`

`make docker-push DOCKER_IMAGE=<some-registry>/<project-name>:tag`
This must be a container registry accessible to your k8 cluster

If using kind for local development, you can replace this with
`kind load docker-image <some-registry>/<project-name>:tag`

Finally to deploy the image

`make deploy DOCKER_IMAGE=<some-registry>/<project-name>:tag`

Handy shortcut command to run these steps at once
`export DOCKER_IMAGE=newrelic/kubernetes-operator:snapshot && make docker-build && kind load docker-image $DOCKER_IMAGE && make deploy`

The newrelic-kubernetes-operator should now be running in your kubernetes cluster.

As an alternative to running the operator in the Kubernetes cluster, you can run the operator locally with `make run`.

# Using the operator

The operator will create and update conditions as needed by applying yaml files with `kubectl apply -f <filename>`

Sample yaml file
```
apiVersion: nr.k8s.newrelic.com/v1
kind: NrqlAlertCondition
metadata:
  name: my-alert
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
  api_key_secret:
    name: nr-api-key
    namespace: default
    key_name: api-key
  region: "us"
```

Please note the `existing_policy_id` field which must be set to a currently existing policy ID in the account configured

`kubectl describe nrqlalertconditions` - describes currently configured alert conditions



# Uninstall the operator

``` bash
$ make uninstall

$ kubectl api-resources > resources-uninstalled.txt

$ diff -u resources-installed.txt resources-uninstalled.txt
```


**Confirm the operator was removed:**

``` diff
--- resources-installed.txt	2020-01-23 12:55:53.000000000 -0700
+++ resources-uninstalled.txt	2020-01-23 12:56:23.000000000 -0700
@@ -40,7 +40,6 @@
 ingresses        # New Relic Kubernetes Operator

[![CircleCI](https://circleci.com/gh/newrelic/newrelic-kubernetes-operator.svg?style=svg)](https://circleci.com/gh/newrelic/newrelic-kubernetes-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/newrelic/newrelic-cli?style=flat-square)](https://goreportcard.com/report/github.com/newrelic/newrelic-kubernetes-operator)
[![GoDoc](https://godoc.org/github.com/newrelic/newrelic-kubernetes-operator?status.svg)](https://godoc.org/github.com/newrelic/newrelic-kubernetes-operator)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/newrelic/newrelic-kubernetes-operator/blob/master/LICENSE)
[![CLA assistant](https://cla-assistant.io/readme/badge/newrelic/newrelic-kubernetes-operator)](https://cla-assistant.io/newrelic/newrelic-kubernetes-operator)
[![Release](https://img.shields.io/github/release/newrelic/newrelic-kubernetes-operator/all.svg)](https://github.com/newrelic/newrelic-kubernetes-operator/releases/latest)

Operator to manage New Relic resources
