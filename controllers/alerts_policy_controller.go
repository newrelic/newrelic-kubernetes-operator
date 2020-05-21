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
	"fmt"
	"strings"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
)

// AlertsPolicyReconciler reconciles a Policy object
type AlertsPolicyReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	AlertClientFunc func(string, string) (interfaces.NewRelicAlertsClient, error)
	apiKey          string
	Alerts          interfaces.NewRelicAlertsClient
	ctx             context.Context
}

// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=policies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=policies/status,verbs=get;update;patch

func (r *AlertsPolicyReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	r.ctx = context.Background()
	_ = r.Log.WithValues("policy", req.NamespacedName)

	var policy nrv1.AlertsPolicy
	err := r.Client.Get(r.ctx, req.NamespacedName, &policy)
	if err != nil {
		if strings.Contains(err.Error(), " not found") {
			r.Log.Info("Policy 'not found' after being deleted. This is expected and no cause for alarm", "error", err)
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "Failed to GET policy", "name", req.NamespacedName.String())
		return ctrl.Result{}, nil
	}

	r.Log.Info(fmt.Sprintf("starting reconcile %T", r))
	r.Log.Info("policy", "policy.Spec.Condition", policy.Spec.Conditions, "policy.status.applied.conditions", policy.Status.AppliedSpec.Conditions)

	r.apiKey = r.getAPIKeyOrSecret(policy)

	if r.apiKey == "" {
		return ctrl.Result{}, errors.New("api key is blank")
	}
	//initial alertsClient
	alertsClient, errAlertsClient := r.AlertClientFunc(r.apiKey, policy.Spec.Region)
	if errAlertsClient != nil {
		r.Log.Error(errAlertsClient, "Failed to create AlertsClient")
		return ctrl.Result{}, errAlertsClient
	}
	r.Alerts = alertsClient

	deleteFinalizer := "alertspolicies.finalizers.nr.k8s.newrelic.com"

	//examine DeletionTimestamp to determine if object is under deletion
	if policy.DeletionTimestamp.IsZero() {
		if !containsString(policy.Finalizers, deleteFinalizer) {
			policy.Finalizers = append(policy.Finalizers, deleteFinalizer)
		}
	} else {
		return r.deletePolicy(r.ctx, &policy, deleteFinalizer)
	}

	//if reflect.DeepEqual(&policy.Spec, policy.Status.AppliedSpec) {
	//	return ctrl.Result{}, nil
	//}

	if policy.Spec.Equals(*policy.Status.AppliedSpec) {
		return ctrl.Result{}, nil
	}

	r.Log.Info("reconciling", "policy", policy.Name)

	r.checkForExistingPolicy(&policy)

	if policy.Status.PolicyID != 0 {
		err := r.updatePolicy(&policy)
		if err != nil {
			r.Log.Error(err, "error updating policy")
			return ctrl.Result{}, err
		}
	} else {
		err := r.createPolicy(&policy)
		if err != nil {
			r.Log.Error(err, "error creating policy")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *AlertsPolicyReconciler) createPolicy(policy *nrv1.AlertsPolicy) error {

	p := alerts.AlertsPolicyInput{}
	p.IncidentPreference = alerts.AlertsIncidentPreference(policy.Spec.IncidentPreference)
	p.Name = policy.Spec.Name

	// p := policy.Spec.APIAlertsPolicy()

	r.Log.Info("Creating policy", "PolicyName", p.Name)
	createResult, err := r.Alerts.CreatePolicyMutation(policy.Spec.AccountID, p)
	if err != nil {
		r.Log.Error(err, "failed to create policy via New Relic API",
			"policyId", policy.Status.PolicyID,
			"region", policy.Spec.Region,
			"Api Key", interfaces.PartialAPIKey(r.apiKey),
		)
		return err
	}
	policy.Status.PolicyID = createResult.ID

	// err = ensureConditions(policy.Spec.Conditions)

	// createdPolicy, err := r.Alerts.CreatePolicy(APIPolicy)
	// if err != nil {
	// 	r.Log.Error(err, "failed to create policy via New Relic API",
	// 		"policyId", policy.Status.PolicyID,
	// 		"region", policy.Spec.Region,
	// 		"Api Key", interfaces.PartialAPIKey(r.apiKey),
	// 	)
	// 	return err
	// }
	// policy.Status.PolicyID = createdPolicy.ID
	//
	// errConditions := r.createConditions(policy)
	// if errConditions != nil {
	// 	r.Log.Error(errConditions, "error creating or updating conditions")
	// 	return errConditions
	// }
	// r.Log.Info("policy after condition creation", "policyCondition", policy.Spec.Conditions, "pointer", &policy)

	policy.Status.AppliedSpec = &policy.Spec

	// err = r.Client.Update(r.ctx, policy)
	// if err != nil {
	// 	r.Log.Error(err, "tried updating policy status", "name", policy.Name)
	// 	return err
	// }
	return nil
}

func (r *AlertsPolicyReconciler) updatePolicy(policy *nrv1.AlertsPolicy) error {
	r.Log.Info("updating policy", "PolicyName", policy.Name)

	//only update policy if policy fields have changed
	APIPolicy := policy.Spec.APIAlertsPolicy()
	APIPolicy.ID = policy.Status.PolicyID

	updatedPolicy := alerts.AlertsPolicyUpdateInput{}

	updatedPolicy.Name = policy.Spec.Name
	updatedPolicy.IncidentPreference = alerts.AlertsIncidentPreference(policy.Spec.IncidentPreference)

	updateResult, err := r.Alerts.UpdatePolicyMutation(policy.Spec.AccountID, policy.Status.PolicyID, updatedPolicy)
	if err != nil {
		r.Log.Error(err, "failed to update policy via New Relic API",
			"policyId", policy.Status.PolicyID,
			"region", policy.Spec.Region,
			"apiKey", interfaces.PartialAPIKey(r.apiKey),
		)
		return err
	}

	policy.Status.PolicyID = updateResult.ID

	// if string(APIPolicy.IncidentPreference) != policy.Status.AppliedSpec.IncidentPreference || APIPolicy.Name != policy.Status.AppliedSpec.Name {
	// 	r.Log.Info("need to update alert policy via New Relic API",
	// 		"Alert Policy Name", APIPolicy.Name,
	// 		"incident preference ", policy.Status.AppliedSpec.IncidentPreference,
	// 	)
	// 	updatedPolicy, err = r.Alerts.UpdatePolicy(APIAlertsPolicy)
	// 	if err != nil {
	// 		r.Log.Error(err, "failed to update policy via New Relic API",
	// 			"policyId", policy.Status.PolicyID,
	// 			"region", policy.Spec.Region,
	// 			"Api Key", interfaces.PartialAPIKey(r.apiKey),
	// 		)
	// 		return err
	// 	}
	// }

	// errConditions := r.createOrUpdateConditions(policy)
	// if errConditions != nil {
	// 	r.Log.Error(errConditions, "error creating or updating conditions")
	// 	return errConditions
	// }
	// r.Log.Info("policySpecx before update", "policy.Spec", policy.Spec)

	policy.Status.AppliedSpec = &policy.Spec

	// err = r.Client.Update(r.ctx, policy)
	// if err != nil {
	// 	r.Log.Error(err, "failed to update policy status", "name", policy.Name)
	// 	return err
	// }
	return nil
}

func (r *AlertsPolicyReconciler) deletePolicy(ctx context.Context, policy *nrv1.AlertsPolicy, deleteFinalizer string) (ctrl.Result, error) {
	// The object is being deleted
	if containsString(policy.Finalizers, deleteFinalizer) {
		// catch invalid state
		if policy.Status.PolicyID == 0 {
			r.Log.Info("No Condition ID set, just removing finalizer")
			policy.Finalizers = removeString(policy.Finalizers, deleteFinalizer)
		} else {
			// our finalizer is present, so lets handle any external dependency
			// collectedErrors := new(customErrors.ErrorCollector)
			// for _, condition := range policy.Status.AppliedSpec.Conditions {
			// 	err := r.deleteCondition(&condition)
			// 	if err != nil {
			// 		r.Log.Error(err, "error deleting condition resources")
			// 		collectedErrors.Collect(err)
			// 	}
			// }
			// if len(*collectedErrors) > 0 {
			// 	r.Log.Info("errors deleting condition resources", "collectedErrors", collectedErrors)
			// 	return ctrl.Result{}, collectedErrors
			// }

			if err := r.deleteNewRelicAlertPolicy(policy); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				r.Log.Error(err, "Failed to delete Alert Policy via New Relic API",
					"policyId", policy.Status.PolicyID,
					"region", policy.Spec.Region,
					"apiKey", interfaces.PartialAPIKey(r.apiKey),
				)
				return ctrl.Result{}, err
			}
			// remove our finalizer from the list and update it.
			r.Log.Info("New Relic Alert policy deleted, Removing finalizer")
			policy.Finalizers = removeString(policy.Finalizers, deleteFinalizer)
			if err := r.Client.Update(ctx, policy); err != nil {
				r.Log.Error(err, "Failed to update k8s records for this policy after successfully deleting the policy via New Relic Alert API")
				return ctrl.Result{}, err
			}
		}
	}

	// Stop reconciliation as the item is being deleted
	r.Log.Info("All done with policy deletion", "policyName", policy.Spec.Name)

	return ctrl.Result{}, nil
}

func (r *AlertsPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nrv1.AlertsPolicy{}).
		Complete(r)
}

func (r *AlertsPolicyReconciler) checkForExistingPolicy(policy *nrv1.AlertsPolicy) {
	if policy.Status.PolicyID == 0 {
		r.Log.Info("Checking for existing policy", "policy", policy.Name, "policyName", policy.Spec.Name)
		//if no policyId, get list of policies and compare name
		alertParams := &alerts.ListPoliciesParams{
			Name: policy.Spec.Name,
		}
		existingPolicies, err := r.Alerts.ListPolicies(alertParams)
		if err != nil {
			r.Log.Error(err, "failed to get list of policies from New Relic API",
				"policyId", policy.Status.PolicyID,
				"region", policy.Spec.Region,
				"apiKey", interfaces.PartialAPIKey(r.apiKey),
			)
		} else {
			for _, existingPolicy := range existingPolicies {
				if existingPolicy.Name == policy.Spec.Name {
					r.Log.Info("matched on existing policy, updating PolicyId", "policyId", existingPolicy.ID)
					policy.Status.PolicyID = existingPolicy.ID
					break
				}
			}
		}
	}
}

func (r *AlertsPolicyReconciler) deleteNewRelicAlertPolicy(policy *nrv1.AlertsPolicy) error {
	r.Log.Info("deleting policy", "policyName", policy.Spec.Name)
	_, err := r.Alerts.DeletePolicyMutation(policy.Spec.AccountID, policy.Status.PolicyID)
	if err != nil {
		r.Log.Error(err, "error deleting policy via New Relic API",
			"policyId", policy.Status.PolicyID,
			"region", policy.Spec.Region,
			"apiKey", interfaces.PartialAPIKey(r.apiKey),
		)
		return err
	}
	return nil
}

func (r *AlertsPolicyReconciler) getAPIKeyOrSecret(policy nrv1.AlertsPolicy) string {

	if policy.Spec.APIKey != "" {
		return policy.Spec.APIKey
	}
	if policy.Spec.APIKeySecret != (nrv1.NewRelicAPIKeySecret{}) {
		key := types.NamespacedName{Namespace: policy.Spec.APIKeySecret.Namespace, Name: policy.Spec.APIKeySecret.Name}
		var apiKeySecret v1.Secret
		getErr := r.Client.Get(context.Background(), key, &apiKeySecret)
		if getErr != nil {
			r.Log.Error(getErr, "Failed to retrieve secret", "secret", apiKeySecret)
			return ""
		}
		return string(apiKeySecret.Data[policy.Spec.APIKeySecret.KeyName])
	}
	return ""
}
