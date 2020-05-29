[![Community Project header](https://github.com/newrelic/open-source-office/raw/master/examples/categories/images/Community_Project.png)](https://github.com/newrelic/open-source-office/blob/master/examples/categories/index.md#category-community-project)

# New Relic Kubernetes Operator

[![Testing](https://github.com/newrelic/newrelic-kubernetes-operator/workflows/Testing/badge.svg)](https://github.com/newrelic/newrelic-kubernetes-operator)
[![Security Scan](https://github.com/newrelic/newrelic-kubernetes-operator/workflows/Security%20Scan/badge.svg)](https://github.com/newrelic/newrelic-kubernetes-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/newrelic/newrelic-cli?style=flat-square)](https://goreportcard.com/report/github.com/newrelic/newrelic-kubernetes-operator)
[![GoDoc](https://godoc.org/github.com/newrelic/newrelic-kubernetes-operator?status.svg)](https://godoc.org/github.com/newrelic/newrelic-kubernetes-operator)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/newrelic/newrelic-kubernetes-operator/blob/master/LICENSE)
[![CLA assistant](https://cla-assistant.io/readme/badge/newrelic/newrelic-kubernetes-operator)](https://cla-assistant.io/newrelic/newrelic-kubernetes-operator)
[![Release](https://img.shields.io/github/release/newrelic/newrelic-kubernetes-operator/all.svg)](https://github.com/newrelic/newrelic-kubernetes-operator/releases/latest)

[![Docker Stars](https://img.shields.io/docker/stars/newrelic/k8s-operator.svg)](https://hub.docker.com/r/newrelic/k8s-operator)
[![Docker Pulls](https://img.shields.io/docker/pulls/newrelic/k8s-operator.svg)](https://hub.docker.com/r/newrelic/k8s-operator)
[![Docker Size](https://img.shields.io/docker/image-size/newrelic/k8s-operator.svg?sort=semver)](https://hub.docker.com/r/newrelic/k8s-operator)
[![Docker Version](https://img.shields.io/docker/v/newrelic/k8s-operator.svg?sort=semver)](https://hub.docker.com/r/newrelic/k8s-operator)

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Using the Operator to Provision Resources](#provision-new-relic-resources-with-the-operator)
  - [Create a New Relic alert policy](##create-a-new-relic-alert-policy-with-nrql-alert-conditions)
  - [Create a New Relic NRQL alert condition](#create-a-nrql-alert-condition-and-add-it-to-an-existing-alert-policy)
- [Development](#development)


# Overview

The **newrelic-kubernetes-operator** is a [Kubernetes Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) that facilitates management of New Relic resources from within your Kubernetes configuration. Managing New Relic resources via [custom Kubernetes objects](https://github.com/newrelic/newrelic-kubernetes-operator/blob/master/examples/example.yaml#L2) can be done the same way you manage built-in Kubernetes objects.

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

<br>

# Provision New Relic resources with the operator

Once you've completed the [Quick Start](#quick-start) section, you can start provisioning New Relic resources with our New Relic Kubernetes objects.

### Create a New Relic alert policy with NRQL alert conditions

1. We'll be using the following [example policy](/examples/example_policy.yaml) configuration file. You will need to update the [`api_key`](/examples/example_policy.yaml#10) field with your New Relic [personal API key](https://docs.newrelic.com/docs/apis/get-started/intro-apis/types-new-relic-api-keys#personal-api-key). <br>

    **examples/example_policy.yaml**

    ```yaml
    apiVersion: nr.k8s.newrelic.com/v1
    kind: AlertsPolicy
    metadata:
      name: my-policy
    spec:
      # Add your API key here
      api_key: <your New Relic personal API key>
      name: k8s created policy
      incident_preference: "PER_POLICY"
      region: "us"
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

   Once you've added your API key, we can apply it your local cluster.
   ```bash
   kubectl apply -f examples/example_policy.yaml
   ```
   > <small>**Note:** You can also use a [Kubernetes secret](https://kubernetes.io/docs/concepts/configuration/secret/) for providing your API key. We've provided an [example secret](/examples/example_secret.yaml) configuration file in case you want to use this method. You'll need to replace `api_key` with [`api_key_secret`](/examples/example_policy.yaml#11). </small>

2. See your configured policies with the following command.
   ```bash
   kubectl describe policies.nr.k8s.newrelic.com
   ```
   > <small>**Note:** You should also see the newly created policy within your New Relic account.</small>

The operator will create and update alert policies and NRQL alert conditions as needed by applying your configuration files with `kubectl apply -f <filename>`

### Create a NRQL alert condition and add it to an existing alert policy

1. We'll be using the following [example NRQL alert condition](/examples/example_nrql_alert_condition.yaml) configuration file. You will need to update the [`api_key`](/examples/example_nrql_alert_condition.yaml#10) field with your New Relic [personal API key](https://docs.newrelic.com/docs/apis/get-started/intro-apis/types-new-relic-api-keys#personal-api-key). <br>

    **examples/example_nrql_alert_condition.yaml**

    ```yaml
    apiVersion: nr.k8s.newrelic.com/v1
    kind: AlertsNrqlCondition
    metadata:
      name: my-alert-condition
    spec:
      # Add your API key here
      api_key: <your New Relic personal API key>
      name: "K8s generated alert condition"
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
      existing_policy_id: 26458245 # Note: must match an existing policy in your account
      region: "us"
    ```

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

1. Run the test suite, which uses the [Ginkgo](http://onsi.github.io/ginkgo/) testing framework. Using the `make` targets is the quickest way to get started with testing.
    - Running tests with `make`
      ```bash
      make test              # runs all tests
      make test-unit         # only runs unit tests
      make test-integration  # only runs integration tests
      ```
    - Linting the codebase
      ```bash
      make lint
      ```

1. Perform the steps from the [Quick Start](#quick-start) section, which walk through the initial required setup and get you going with your first `kubectl apply` of the operator configuration.

1. Confirm your configuration was deployed to your local Kubernetes cluster (the one that we created with `kind`). <br>
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

1. Next steps:
    - [Create a New Relic alert policy](#create-a-new-relic-alert-policy-with-nrql-alert-conditions)
    - [Create a New Relic NRQL alert condition](#create-a-nrql-alert-condition-and-add-it-to-an-existing-alert-policy)

<br>

## Helpful commands

```bash
# Describe the currently configured policies.
kubectl describe alertspolicies.nr.k8s.newrelic.com

# Describe the currently configured alert conditions.
kubectl describe alertsnrqlconditions.nr.k8s.newrelic.com

# Get the node being used for the newrelic operator.
kubectl get nodes -n newrelic-kubernetes-operator-system

# Describe the node being used for the newrelic operator.
kubectl describe node <your-node-name>

# Tail logs of the operator's manager container (useful during development).
# Use the `describe node` command above to locate your manager controller.
kubectl logs -f -n newrelic-kubernetes-operator-system -c manager newrelic-kubernetes-operator-controller-manager-<hash from>
```
