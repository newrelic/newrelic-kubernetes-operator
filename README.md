# New Relic Kubernetes Operator
Operator to manage New Relic resources

# Development Prerequisites

To get required tooling:

```bash
brew install kubectl kubebuilder kustomize
```

You will also want to install Docker for Mac and enable its built-in kubernetes cluster functionality.


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
+nrqlalertconditions                            nr-alerts.k8s.newrelic.com     true         NrqlAlertCondition
 poddisruptionbudgets              pdb          policy                         true         PodDisruptionBudget
 podsecuritypolicies               psp          policy                         false        PodSecurityPolicy
 clusterrolebindings                            rbac.authorization.k8s.io      false        ClusterRoleBinding
```

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
 ingresses                         ing          networking.k8s.io              true         Ingress
 networkpolicies                   netpol       networking.k8s.io              true         NetworkPolicy
 runtimeclasses                                 node.k8s.io                    false        RuntimeClass
-nrqlalertconditions                            nr-alerts.k8s.newrelic.com     true         NrqlAlertCondition
 poddisruptionbudgets              pdb          policy                         true         PodDisruptionBudget
 podsecuritypolicies               psp          policy                         false        PodSecurityPolicy
 clusterrolebindings                            rbac.authorization.k8s.io      false        ClusterRoleBinding
```
