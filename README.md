[![Community Project header](https://github.com/newrelic/open-source-office/raw/master/examples/categories/images/Community_Project.png)](https://github.com/newrelic/open-source-office/blob/master/examples/categories/index.md#category-community-project)

# New Relic Kubernetes Operator

[![Testing](https://github.com/newrelic/newrelic-kubernetes-operator/workflows/Testing/badge.svg)](https://github.com/newrelic/newrelic-kubernetes-operator)
[![Security Scan](https://github.com/newrelic/newrelic-kubernetes-operator/workflows/Security%20Scan/badge.svg)](https://github.com/newrelic/newrelic-kubernetes-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/newrelic/newrelic-cli?style=flat-square)](https://goreportcard.com/report/github.com/newrelic/newrelic-kubernetes-operator)
[![GoDoc](https://godoc.org/github.com/newrelic/newrelic-kubernetes-operator?status.svg)](https://godoc.org/github.com/newrelic/newrelic-kubernetes-operator)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/newrelic/newrelic-kubernetes-operator/blob/master/LICENSE)
[![CLA assistant](https://cla-assistant.io/readme/badge/newrelic/newrelic-kubernetes-operator)](https://cla-assistant.io/newrelic/newrelic-kubernetes-operator)
[![Release](https://img.shields.io/github/release/newrelic/newrelic-kubernetes-operator/all.svg)](https://github.com/newrelic/newrelic-kubernetes-operator/releases/latest)

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Using the Operator](#using-the-operator)
- [Development](#development)


# Overview

The **newrelic-kubernetes-operator** is a [Kubernetes Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) that facilitates management of New Relic resources from within your Kubernetes configuration. Managing New Relic resources via [custom Kubernetes objects](https://github.com/newrelic/newrelic-kubernetes-operator/blob/master/examples/example.yaml#L2) can be done the same way you manage built-in [Kubernetes objects](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#describing-a-kubernetes-object).

Currently the operator supports managing the following resources:
- Alert Policies
- NRQL Alert Conditions.


# Quick Start
> <small>**Note:** These quick start instructions do **not** require you to clone the repo.</small>

## Running Kubernetes in a Docker container locally with [kind](https://kind.sigs.k8s.io/)

1. Install docker, kubectl, kustomize, and kind

   ```bash
   brew cask install docker
   brew install kubernetes-cli kustomize kind
   ```

1. Create a test cluster with kind

   ```bash
   kind create cluster --name newrelic
   kubectl cluster-info
   ```

1. Install cert-manager

   ```bash
   kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/download/v0.15.0/cert-manager.yaml
   ```

   > <small>**Note:** This takes a minute or two to finish so wait a minute before going on to the next step.</small>

   You can also confirm it's running with the command `kubectl rollout status deployment -n cert-manager cert-manager-webhook`

1. Install the operator in the test cluster.

   ```bash
   kustomize build github.com/newrelic/newrelic-kubernetes-operator/configs/default | kubectl apply -f -
   ```
   > <small>**Note:** This will install operator on whatever kubernetes cluster kubectl is configured to use.</small>

## Using a custom container

If you want to deploy the operator in a custom container you can override the image name with a `kustomize.yaml` file.

1. Create a new `kustomize.yaml` file

   ```yaml
   apiVersion: kustomize.config.k8s.io/v1beta1
   kind: Kustomization
   namespace: newrelic-kubernetes-operator-system
   resources:
     - github.com/newrelic/newrelic-kubernetes-operator/configs/default
   images:
     - name: newrelic/k8s-operator:snapshot
       newName: <CUSTOM_IMAGE>
       newTag: <CUSTOM_TAG>
   ```

1. The apply the file with:

   ```bash
   kustomize build . | kubectl apply -f -
   ```

# Using the operator

The operator will create and update alert policies and NRQL alert conditions as needed by applying yaml files with `kubectl apply -f <filename>`

#### Create an alert policy with a NRQL alert condition
```yaml
apiVersion: nr.k8s.newrelic.com/v1
kind: Policy
metadata:
  name: my-policy
spec:
  name: k8s created policy
  incident_preference: "PER_POLICY"
  region: "us"
  # API_KEY can be specified directly in the yaml file or via a k8s secret
  #api_key: APIKEY
  api_key_secret:
    name: nr-api-key
    namespace: default
    key_name: api-key
  conditions:
    - spec:
        nrql:
          # Note: This is just an example.
          # You'll want to use a query with parameters that are
          # more specific to the needs for targeting associated
          # kubernetes objects.
          query: "SELECT count(*) FROM Transactions WHERE ..."
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
          query: "SELECT count(*) FROM Transactions WHERE ..."
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
<br>

#### Create a NRQL alert condition and add it to an existing alert policy
```yaml
apiVersion: nr.k8s.newrelic.com/v1
kind: NrqlAlertCondition
metadata:
  name: my-alert-condition
spec:
  nrql:
    # Note: This is just an example.
    # You'll want to use a query with parameters that are
    # more specific to the needs for targeting associated
    # kubernetes objects.
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
  existing_policy_id: 26458245 # Note: this must match an existing policy in your account
  region: "us"
  # API_KEY can be specified directly in the yaml file or via a k8s secret
  #api_key: API_KEY
  api_key_secret:
    name: nr-api-key
    namespace: default
    key_name: api-key
```

### Helpful commands

- `kubectl describe nrqlalertconditions.nr.k8s.newrelic.com` - describes currently configured alert conditions

- `kubectl describe policies.nr.k8s.newrelic.com` - describes currently configured alert conditions

### Uninstall the operator

The Operator can be removed with the reverse of installation, namely building the kubernetes resource files with `kustomize` and running `kubectl delete`

```bash
kustomize build github.com/newrelic/newrelic-kubernetes-operator/configs/default | kubectl delete -f -
```
<br>

# Development

This section should get you set up properly for doing development on the operator.

#### Requirements
- [Go](https://golang.org/) v1.13+
- [Docker](https://www.docker.com/get-started) (with Kubernetes enabled)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [kustomize](https://kustomize.io/)
- [kubebuilder](https://book.kubebuilder.io/quick-start.html)

#### Code
1. Clone the repo
    ```bash
    git clone git@github.com:newrelic/newrelic-kubernetes-operator.git
    ```

1. Install [kubebuilder](https://go.kubebuilder.io/quick-start.html#prerequisites) following the instructions for your operating system. This installation will also get `etcd` and `kube-apiserver` which are needed for the tests. <br>
    > <small>**Note:** Do **_not_** install `kubebuilder` with `brew`. Homebrew's `kubebuilder` package will not provide all the necessary dependencies for running the tests.</small>

1. Run the test suite, which uses the [Ginkgo](http://onsi.github.io/ginkgo/) testing framework. Executing tests can be done in a few ways, but using the `make` targets is the quickest way to get started with testing.
    - Running tests with `make`
      ```bash
      make test              # runs all tests
      make test-unit         # only runs unit tests
      make test-integration  # only runs integration tests
      ```
    - Running tests with `ginkgo`
      ```bash
      ginkgo --tags unit -r ./          # only runs unit tests
      ginkgo --tags integration -r ./   # only runs integrations tests
      ```
    - Linting the codebase
      ```bash
      make lint
      ```
1. Perform the steps from the [Quick Start](#quick-start) section, which walk through the initial required setup and get you going with your first `kubectl apply` of the operator configuration.

1. Confirm your configuration was deployed to your local kubernetes cluster (the one that we created with `kind`). <br>
    - Show your namespaces. You should see `newrelic-kubernetes-operator-system` in the list of namespaces.
      ```bash
      kubectl get namespaces
      ```
    - Show the nodes within the `newrelic-kubernetes-operator-system` namespace.
      ```bash
      kubectl get nodes -n newrelic-kubernetes-operator-system
      ```
      You should see something similar to the following output:
      ```
      NAME                     STATUS   ROLES    AGE    VERSION
      newrelic-control-plane   Ready    master   163m   v1.18.2
      ```

1. Now we can try creating a New Relic alert policy with an [example config](/examples/example_policy.yaml). You will need to update the [`api_key`](/examples/example_policy.yaml#10) field with your New Relic [personal API key](https://docs.newrelic.com/docs/apis/get-started/intro-apis/types-new-relic-api-keys#personal-api-key).
   ```bash
   kubectl apply -f examples/examply_policy.yaml
   ```
   ```bash

   ```
   > <small>**Note:** Secrets management for your New Relic personal API key can also be referenced as [Kubernetes secret](https://kubernetes.io/docs/concepts/configuration/secret/). We've provided an [example secret config](/examples/example_secret.yaml) file for you in case you want to use this method.</small>
