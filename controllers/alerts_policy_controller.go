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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	customErrors "github.com/newrelic/newrelic-kubernetes-operator/errors"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
)

// AlertsPolicyReconciler reconciles a AlertsPolicy object
type AlertsPolicyReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	AlertClientFunc func(string, string) (interfaces.NewRelicAlertsClient, error)
	apiKey          string
	Alerts          interfaces.NewRelicAlertsClient
	ctx             context.Context
}

// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=alertspolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=alertspolicies/status,verbs=get;update;patch

func (r *AlertsPolicyReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	r.ctx = context.Background()
	_ = r.Log.WithValues("policy to reconcile", req.NamespacedName)

	var policy nrv1.AlertsPolicy
	err := r.Client.Get(r.ctx, req.NamespacedName, &policy)
	if err != nil {
		if strings.Contains(err.Error(), " not found") {
			r.Log.Info("AlertsPolicy 'not found' after being deleted. This is expected and no cause for alarm", "error", err)
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "failed to GET policy", "name", req.NamespacedName.String())
		return ctrl.Result{}, nil
	}
	r.Log.Info("starting reconcile action")
	r.Log.Info("policy", "policy.Spec.Condition", policy.Spec.Conditions, "policy.status.applied.conditions", policy.Status.AppliedSpec.Conditions)

	r.apiKey = r.getAPIKeyOrSecret(policy)

	if r.apiKey == "" {
		return ctrl.Result{}, errors.New("api key is blank")
	}
	//initial alertsClient
	alertsClient, errAlertsClient := r.AlertClientFunc(r.apiKey, policy.Spec.Region)
	if errAlertsClient != nil {
		r.Log.Error(errAlertsClient, "failed to create AlertsClient")
		return ctrl.Result{}, errAlertsClient
	}
	r.Alerts = alertsClient

	deleteFinalizer := "policies.finalizers.nr.k8s.newrelic.com"

	//examine DeletionTimestamp to determine if object is under deletion
	if policy.DeletionTimestamp.IsZero() {
		if !containsString(policy.Finalizers, deleteFinalizer) {
			policy.Finalizers = append(policy.Finalizers, deleteFinalizer)
		}
	} else {
		return r.deleteAlertsPolicy(r.ctx, &policy, deleteFinalizer)
	}

	//if reflect.DeepEqual(&policy.Spec, policy.Status.AppliedSpec) {
	//	return ctrl.Result{}, nil
	//}

	if policy.Spec.Equals(*policy.Status.AppliedSpec) {
		return ctrl.Result{}, nil
	}

	r.Log.Info("Reconciling", "policy", policy.Name)

	r.checkForExistingAlertsPolicy(&policy)

	if policy.Status.PolicyID != 0 {
		err := r.updateAlertsPolicy(&policy)
		if err != nil {
			r.Log.Error(err, "error updating policy")
			return ctrl.Result{}, err
		}
	} else {
		err := r.createAlertsPolicy(&policy)
		if err != nil {
			r.Log.Error(err, "error creating policy")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *AlertsPolicyReconciler) createAlertsPolicy(policy *nrv1.AlertsPolicy) error {
	p := alerts.AlertsPolicyInput{}
	p.IncidentPreference = alerts.AlertsIncidentPreference(policy.Spec.IncidentPreference)
	p.Name = policy.Spec.Name

	r.Log.Info("creating policy", "PolicyName", p.Name)
	createResult, err := r.Alerts.CreatePolicyMutation(policy.Spec.AccountID, p)
	if err != nil {
		r.Log.Error(err, "failed to create policy via New Relic API",
			"policyId", policy.Status.PolicyID,
			"region", policy.Spec.Region,
			"apiKey", interfaces.PartialAPIKey(r.apiKey),
		)
		return err
	}
	policy.Status.PolicyID = createResult.ID

	errConditions := r.createConditions(policy)
	if errConditions != nil {
		r.Log.Error(errConditions, "error creating or updating conditions")
		return errConditions
	}
	r.Log.Info("policy after condition creation", "policyCondition", policy.Spec.Conditions, "pointer", &policy)

	policy.Status.AppliedSpec = &policy.Spec

	err = r.Client.Update(r.ctx, policy)
	if err != nil {
		r.Log.Error(err, "tried updating policy status", "name", policy.Name)
		return err
	}

	return nil
}

func (r *AlertsPolicyReconciler) createConditions(policy *nrv1.AlertsPolicy) error {
	r.Log.Info("creating conditions for policy")

	collectedErrors := new(customErrors.ErrorCollector)

	for i, condition := range policy.Spec.Conditions {
		err := r.createCondition(policy, &condition)
		if err != nil {
			r.Log.Error(err, "error creating condition")
			collectedErrors.Collect(err)
		} else {
			policy.Spec.Conditions[i] = condition
		}
	}

	if len(*collectedErrors) > 0 {
		r.Log.Info("errors encountered creating conditions", "collectoredErrors", collectedErrors)
		return collectedErrors
	}

	return nil
}

type processedAlertsNrqlConditions struct {
	processed bool
	condition nrv1.AlertsPolicyCondition
}

func (r *AlertsPolicyReconciler) createOrUpdateConditions(policy *nrv1.AlertsPolicy) error {
	if reflect.DeepEqual(policy.Spec.Conditions, policy.Status.AppliedSpec.Conditions) {
		return nil
	}

	//build map of existing conditions so we can mark them off as processed and delete anything left over
	var existingConditions = make(map[string]processedAlertsNrqlConditions)
	for _, existing := range policy.Status.AppliedSpec.Conditions {
		existingConditions[existing.Name] = processedAlertsNrqlConditions{
			processed: false,
			condition: existing,
		}
	}

	collectedErrors := new(customErrors.ErrorCollector)

	for i, condition := range policy.Spec.Conditions {
		//loop through the policies, creating/updating as needed
		r.Log.Info("checking on condition", "resourceName", condition.Name, "conditionName", condition.Spec.Name)
		//first we check to see if the name is set
		if condition.Name == "" {
			//If resource name is not set, let's see if a appliedSpec matches the NR condition name
			r.Log.Info("condition name not set, checking name values")
			for _, appliedCondition := range policy.Status.AppliedSpec.Conditions {

				if appliedCondition.Spec.Name == condition.Spec.Name {
					r.Log.Info("Found matching name")
					condition.Namespace = appliedCondition.Namespace
					condition.Name = appliedCondition.Name
					break
				}
			}

			if condition.Name == "" {
				r.Log.Info("made it through all existing appliedConditions, creating a new one")
				err := r.createCondition(policy, &condition)
				if err != nil {
					r.Log.Error(err, "error creating condition")
					collectedErrors.Collect(err)
				}

				//add to the list of processed conditions
				existingConditions[condition.Name] = processedAlertsNrqlConditions{
					processed: true,
					condition: condition,
				}
				//Now update the spec
				policy.Spec.Conditions[i] = condition
				r.Log.Info("policy spec updated", "policy.Spec.Condition[i]", policy.Spec.Conditions[i])
				//move to the next
				continue
			}
		}

		r.Log.Info("now we have a condition name", "conditionName", condition.Name)
		existingConditions[condition.Name] = processedAlertsNrqlConditions{
			processed: true,
			condition: condition,
		}

		retrievedCondition := r.getAlertsNrqlConditionFromAlertsPolicyCondition(&condition)

		r.Log.Info("found condition to update", "retrievedCondition", retrievedCondition)

		//Now check to confirm the NrqlCondition matches out AlertsPolicyCondition
		retrievedAlertsPolicyCondition := nrv1.AlertsPolicyCondition{Spec: retrievedCondition.Spec}
		r.Log.Info("spec hash", "retrieved", retrievedAlertsPolicyCondition.SpecHash(), "condition", condition.SpecHash())
		r.Log.Info("conditions", "retrieved", retrievedAlertsPolicyCondition, "condition", condition)

		if retrievedAlertsPolicyCondition.SpecHash() == condition.SpecHash() {
			r.Log.Info("existing NrqlCondition matches going to next")
			policy.Spec.Conditions[i] = condition
			continue
		}
		r.Log.Info("updating existing condition", "policyRegion", policy.Spec.Region, "policyId", policy.Status.PolicyID)

		retrievedCondition.Spec = condition.Spec
		//Set inherited values
		retrievedCondition.Spec.Region = policy.Spec.Region
		retrievedCondition.Spec.ExistingPolicyID = policy.Status.PolicyID
		retrievedCondition.Spec.APIKey = policy.Spec.APIKey
		retrievedCondition.Spec.APIKeySecret = policy.Spec.APIKeySecret

		err := r.Client.Update(r.ctx, &retrievedCondition)
		if err != nil {
			r.Log.Error(err, "error updating condition")
			collectedErrors.Collect(err)
		}
		//Now update the spec
		policy.Spec.Conditions[i] = condition
		r.Log.Info("policy spec updated", "policy.Spec.Condotion[i]", policy.Spec.Conditions[i])

	}

	r.Log.Info("now one last check for stragglers")
	//now we check for any left behind conditions that weren't processed

	for conditionName, processedCondition := range existingConditions {
		r.Log.Info("checking "+processedCondition.condition.Name, "bool is", processedCondition.processed)
		if !processedCondition.processed {
			r.Log.Info("need to delete", "ppliedConditionName", conditionName)
			err := r.deleteCondition(&processedCondition.condition)
			if err != nil {
				r.Log.Error(err, "error deleting condition resource")
				collectedErrors.Collect(err)
			}
		}

	}
	if len(*collectedErrors) > 0 {
		r.Log.Info("Errors encountered processing conditions", "collectedErrors", collectedErrors)
		return collectedErrors
	}

	r.Log.Info("all done", "policy.Spec", policy.Spec, "policy.Status.AppliedSpec.Conditions", policy.Status.AppliedSpec.Conditions)

	return nil
}

func (r *AlertsPolicyReconciler) createCondition(policy *nrv1.AlertsPolicy, condition *nrv1.AlertsPolicyCondition) error {
	var alertsNrqlCondition nrv1.AlertsNrqlCondition
	alertsNrqlCondition.GenerateName = policy.Name + "-condition-"
	alertsNrqlCondition.Namespace = policy.Namespace
	alertsNrqlCondition.Labels = policy.Labels
	//TODO: no clue if this is needed, I'm guessing no
	//condition.OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(policy, conditionKind)},

	alertsNrqlCondition.Spec = condition.Spec
	alertsNrqlCondition.Spec.Region = policy.Spec.Region
	alertsNrqlCondition.Spec.ExistingPolicyID = policy.Status.PolicyID
	alertsNrqlCondition.Spec.APIKey = policy.Spec.APIKey
	alertsNrqlCondition.Spec.APIKeySecret = policy.Spec.APIKeySecret
	alertsNrqlCondition.Status.AppliedSpec = &nrv1.AlertsNrqlConditionSpec{}

	r.Log.Info("creating condition", "condition", condition.Name, "conditionName", condition.Spec.Name, "nrqlAlertCondition", alertsNrqlCondition)

	errCondition := r.Create(r.ctx, &alertsNrqlCondition)
	if errCondition != nil {
		r.Log.Error(errCondition, "error creating condition")
		return errCondition
	}

	condition.Name = alertsNrqlCondition.Name //created from generated name
	condition.Namespace = alertsNrqlCondition.Namespace
	//condition.SpecHash = nrv1.ComputeHash(&condition.Spec)

	r.Log.Info("created condition", "condition", condition.Name, "conditionName", condition.Spec.Name, "nrqlAlertCondition", alertsNrqlCondition)

	return nil
}

func (r *AlertsPolicyReconciler) deleteCondition(condition *nrv1.AlertsPolicyCondition) error {
	r.Log.Info("deleting condition", "condition", condition.Name, "conditionName", condition.Spec.Name)
	retrievedCondition := r.getAlertsNrqlConditionFromAlertsPolicyCondition(condition)
	r.Log.Info("retrieved condition for deletion", "retrievedCondition", retrievedCondition)
	err := r.Delete(r.ctx, &retrievedCondition)
	if err != nil {
		r.Log.Error(err, "error deleting condition resource")
		return err
	}
	return nil
}

func (r *AlertsPolicyReconciler) getAlertsNrqlConditionFromAlertsPolicyCondition(condition *nrv1.AlertsPolicyCondition) (alertsNrqlCondition nrv1.AlertsNrqlCondition) {
	r.Log.Info("condition before retrieval", "condition", condition)
	//throw away the error since empty conditions are expected
	_ = r.Client.Get(r.ctx, condition.GetNamespace(), &alertsNrqlCondition)
	r.Log.Info("retrieved condition", "alertsNrqlCondition", alertsNrqlCondition, "namespace", condition.GetNamespace())
	return
}

func (r *AlertsPolicyReconciler) updateAlertsPolicy(policy *nrv1.AlertsPolicy) error {
	r.Log.Info("updating policy", "PolicyName", policy.Name)

	//only update policy if policy fields have changed
	APIPolicy := policy.Spec.APIAlertsPolicy()
	APIPolicy.ID = policy.Status.PolicyID
	var updateResult *alerts.Policy
	var err error

	if string(APIPolicy.IncidentPreference) != policy.Status.AppliedSpec.IncidentPreference || APIPolicy.Name != policy.Status.AppliedSpec.Name {
		r.Log.Info("need to update alert policy via New Relic API",
			"alerts policy Name", APIPolicy.Name,
			"incident preference ", policy.Status.AppliedSpec.IncidentPreference,
		)
		updateResult, err = r.Alerts.UpdatePolicy(APIPolicy)
		if err != nil {
			r.Log.Error(err, "failed to update policy via New Relic API",
				"policyId", policy.Status.PolicyID,
				"region", policy.Spec.Region,
				"apiKey", interfaces.PartialAPIKey(r.apiKey),
			)
			return err
		}
		policy.Status.PolicyID = updateResult.ID
	}

	errConditions := r.createOrUpdateConditions(policy)
	if errConditions != nil {
		r.Log.Error(errConditions, "error creating or updating conditions")
		return errConditions
	}
	r.Log.Info("policySpec before update", "policy.Spec", policy.Spec)

	policy.Status.AppliedSpec = &policy.Spec

	err = r.Client.Update(r.ctx, policy)
	if err != nil {
		r.Log.Error(err, "failed to update policy status", "name", policy.Name)
		return err
	}

	return nil
}

func (r *AlertsPolicyReconciler) deleteAlertsPolicy(ctx context.Context, policy *nrv1.AlertsPolicy, deleteFinalizer string) (ctrl.Result, error) {
	// The object is being deleted
	if containsString(policy.Finalizers, deleteFinalizer) {
		// catch invalid state
		if policy.Status.PolicyID == 0 {
			r.Log.Info("no PolicyID set, removing finalizer")
			policy.Finalizers = removeString(policy.Finalizers, deleteFinalizer)
		} else {
			// our finalizer is present, so lets handle any external dependency
			collectedErrors := new(customErrors.ErrorCollector)
			for _, condition := range policy.Status.AppliedSpec.Conditions {
				err := r.deleteCondition(&condition)
				if err != nil {
					r.Log.Error(err, "error deleting condition resources")
					collectedErrors.Collect(err)
				}
			}
			if len(*collectedErrors) > 0 {
				r.Log.Info("errors deleting condition resources", "collectedErrors", collectedErrors)
				return ctrl.Result{}, collectedErrors
			}

			if err := r.deleteNewRelicAlertPolicy(policy); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				r.Log.Error(err, "failed to delete alerts policy via New Relic API",
					"policyId", policy.Status.PolicyID,
					"region", policy.Spec.Region,
					"apiKey", interfaces.PartialAPIKey(r.apiKey),
				)
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			r.Log.Info("removing finalizer")
			policy.Finalizers = removeString(policy.Finalizers, deleteFinalizer)
			if err := r.Client.Update(ctx, policy); err != nil {
				r.Log.Error(err, "failed to update k8s records for this policy after successfully deleting the policy via New Relic Alert API")
				return ctrl.Result{}, err
			}
		}
	}

	// Stop reconciliation as the item is being deleted
	r.Log.Info("policy deletion complete", "policyName", policy.Spec.Name)

	return ctrl.Result{}, nil
}

func (r *AlertsPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nrv1.AlertsPolicy{}).
		Complete(r)
}

func (r *AlertsPolicyReconciler) checkForExistingAlertsPolicy(policy *nrv1.AlertsPolicy) {
	if policy.Status.PolicyID == 0 {
		r.Log.Info("checking for existing policy", "policy", policy.Name, "policyName", policy.Spec.Name)
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
			for _, existingAlertsPolicy := range existingPolicies {
				if existingAlertsPolicy.Name == policy.Spec.Name {
					r.Log.Info("matched on existing policy, updating PolicyId", "policyId", existingAlertsPolicy.ID)
					policy.Status.PolicyID = existingAlertsPolicy.ID
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
			r.Log.Error(getErr, "failed to retrieve secret", "secret", apiKeySecret)
			return ""
		}
		return string(apiKeySecret.Data[policy.Spec.APIKeySecret.KeyName])
	}
	return ""
}
