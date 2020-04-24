/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
)

// PolicyReconciler reconciles a Policy object
type PolicyReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	AlertClientFunc func(string, string) (interfaces.NewRelicAlertsClient, error)
	apiKey          string
	Alerts          interfaces.NewRelicAlertsClient
}

// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=policies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=policies/status,verbs=get;update;patch

func (r *PolicyReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("policy", req.NamespacedName)

	// your logic here

	var policy nrv1.Policy
	err := r.Client.Get(ctx, req.NamespacedName, &policy)
	if err != nil {
		if strings.Contains(err.Error(), " not found") {
			r.Log.Info("Expected error 'not found' after policy deleted", "error", err)
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "Tried getting policy", "name", req.NamespacedName.String())
		return ctrl.Result{}, nil
	}
	r.Log.Info("Starting reconcile action")

	r.apiKey = r.getAPIKeyOrSecret(policy)

	if r.apiKey == "" {
		return ctrl.Result{}, errors.New("api key is blank")
	}
	//initial alertsClient
	alertsClient, errAlertsClient := r.AlertClientFunc(r.apiKey, policy.Spec.Region)
	if errAlertsClient != nil {
		r.Log.Error(errAlertsClient, "Error thrown")
		return ctrl.Result{}, errAlertsClient
	}
	r.Alerts = alertsClient

	deleteFinalizer := "storage.finalizers.tutorial.kubebuilder.io"

	//examine DeletionTimestamp to determine if object is under deletion
	if policy.DeletionTimestamp.IsZero() {
		if !containsString(policy.Finalizers, deleteFinalizer) {
			policy.Finalizers = append(policy.Finalizers, deleteFinalizer)
		}
	} else {
		// The object is being deleted
		if containsString(policy.Finalizers, deleteFinalizer) {
			// catch invalid state
			if policy.Status.PolicyID == 0 {
				r.Log.Info("No Condition ID set, just removing finalizer")
				policy.Finalizers = removeString(policy.Finalizers, deleteFinalizer)
			} else {
				// our finalizer is present, so lets handle any external dependency
				if err := r.deleteNewRelicAlertPolicy(policy); err != nil {
					// if fail to delete the external dependency here, return with error
					// so that it can be retried
					r.Log.Error(err, "Failed to delete API Condition",
						"policyId", policy.Status.PolicyID,
						"region", policy.Spec.Region,
						"Api Key", interfaces.PartialAPIKey(r.apiKey),
					)
					return ctrl.Result{}, err
				}
				// remove our finalizer from the list and update it.
				r.Log.Info("New Relic Alert policy deleted, Removing finalizer")
				policy.Finalizers = removeString(policy.Finalizers, deleteFinalizer)
				if err := r.Client.Update(ctx, &policy); err != nil {
					r.Log.Error(err, "Failed to update policy after deleting New Relic Alert policy")
					return ctrl.Result{}, err
				}
			}
		}

		// Stop reconciliation as the item is being deleted
		r.Log.Info("All done with policy deletion", "policyName", policy.Spec.Name)

		return ctrl.Result{}, nil
	}

	if reflect.DeepEqual(&policy.Spec, policy.Status.AppliedSpec) {
		return ctrl.Result{}, nil
	}

	r.Log.Info("Reconciling", "policy", policy.Name)

	//check if policy has policy id
	r.checkForExistingPolicy(&policy)

	APIPolicy := policy.Spec.APIPolicy()

	if policy.Status.PolicyID != 0 && !reflect.DeepEqual(&policy.Spec, policy.Status.AppliedSpec) {
		r.Log.Info("updating policy", "PolicyName", policy.Name, "API fields", APIPolicy)
		APIPolicy.ID = policy.Status.PolicyID
		updatedCondition, err := alertsClient.UpdatePolicy(APIPolicy)
		if err != nil {
			r.Log.Error(err, "failed to update policy")
		} else {
			policy.Status.AppliedSpec = &policy.Spec
			policy.Status.PolicyID = updatedCondition.ID
		}

		err = r.Client.Update(ctx, &policy)
		if err != nil {
			r.Log.Error(err, "tried updating policy status", "name", req.NamespacedName)
		}
	} else {
		r.Log.Info("Creating policy", "ConditionName", policy.Name, "API fields", APIPolicy)
		createdPolicy, err := alertsClient.CreatePolicy(APIPolicy)
		if err != nil {
			r.Log.Error(err, "failed to create policy",
				"policyId", policy.Status.PolicyID,
				"region", policy.Spec.Region,
				"Api Key", interfaces.PartialAPIKey(r.apiKey),
			)
		} else {
			policy.Status.AppliedSpec = &policy.Spec
			policy.Status.PolicyID = createdPolicy.ID
		}

		err = r.Client.Update(ctx, &policy)
		if err != nil {
			r.Log.Error(err, "tried updating policy status", "name", req.NamespacedName)
		}
	}

	return ctrl.Result{}, nil
}

func (r *PolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nrv1.Policy{}).
		Complete(r)
}

func (r *PolicyReconciler) checkForExistingPolicy(policy *nrv1.Policy) {
	if policy.Status.PolicyID == 0 {
		r.Log.Info("Checking for existing policy", "policyName", policy.Name)
		//if no policyId, get list of policys and compare name
		alertParams := &alerts.ListPoliciesParams{
			Name: "STUB=name", //TODO figure out what this should look like
		}
		existingPolicyies, err := r.Alerts.ListPolicies(alertParams)
		if err != nil {
			r.Log.Error(err, "failed to get list of NRQL policys from New Relic API",
				"policyId", policy.Status.PolicyID,
				"region", policy.Spec.Region,
				"Api Key", interfaces.PartialAPIKey(r.apiKey),
			)
		} else {
			for _, existingPolicy := range existingPolicyies {
				if existingPolicy.Name == policy.Spec.Name {
					r.Log.Info("Matched on existing policy, updating PolicyId", "policyId", existingPolicy.ID)
					policy.Status.PolicyID = existingPolicy.ID
					break
				}
			}
		}
	}
}

func (r *PolicyReconciler) deleteNewRelicAlertPolicy(policy nrv1.Policy) error {
	r.Log.Info("Deleting policy", "policyName", policy.Spec.Name)
	_, err := r.Alerts.DeletePolicy(policy.Status.PolicyID)
	if err != nil {
		r.Log.Error(err, "Error deleting policy",
			"policyId", policy.Status.PolicyID,
			"region", policy.Spec.Region,
			"Api Key", interfaces.PartialAPIKey(r.apiKey),
		)
		return err
	}
	return nil
}

func (r *PolicyReconciler) getAPIKeyOrSecret(policy nrv1.Policy) string {

	if policy.Spec.APIKey != "" {
		return policy.Spec.APIKey
	}
	if policy.Spec.APIKeySecret != (nrv1.NewRelicAPIKeySecret{}) {
		key := types.NamespacedName{Namespace: policy.Spec.APIKeySecret.Namespace, Name: policy.Spec.APIKeySecret.Name}
		var apiKeySecret v1.Secret
		getErr := r.Client.Get(context.Background(), key, &apiKeySecret)

		r.Log.Error(getErr, "Error retrieving secret", "secret", apiKeySecret)
		return string(apiKeySecret.Data[policy.Spec.APIKeySecret.KeyName])
	}
	return ""
}
