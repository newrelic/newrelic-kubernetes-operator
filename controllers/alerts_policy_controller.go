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

	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	customErrors "github.com/newrelic/newrelic-kubernetes-operator/errors"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
)

const (
	alertsPolicyDeleteFinalizer = "policies.finalizers.nr.k8s.newrelic.com"
)

var trueVar = true

// AlertsPolicyReconciler reconciles a AlertsPolicy object
type AlertsPolicyReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	AlertClientFunc func(string, string) (interfaces.NewRelicAlertsClient, error)
	apiKey          string
	Alerts          interfaces.NewRelicAlertsClient
	ctx             context.Context
	NewRelicAgent   newrelic.Application
	txn             *newrelic.Transaction
}

// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=alertspolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=alertspolicies/status,verbs=get;update;patch

func (r *AlertsPolicyReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	r.ctx = context.Background()
	_ = r.Log.WithValues("policy", req.NamespacedName)
	r.txn = r.NewRelicAgent.StartTransaction("Reconcile/Alerts/AlertsPolicy")
	defer r.txn.End()

	var policy nrv1.AlertsPolicy
	err := r.Client.Get(r.ctx, req.NamespacedName, &policy)
	if err != nil {
		if strings.Contains(err.Error(), " not found") {
			r.Log.Info("AlertsPolicy 'not found' after being deleted. This is expected and no cause for alarm", "error", err)
			return ctrl.Result{}, nil
		}

		r.Log.Error(err, "Failed to GET policy", "name", req.NamespacedName.String())
		return ctrl.Result{}, nil
	}
	r.Log.Info("Starting reconcile action")
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

	//examine DeletionTimestamp to determine if object is under deletion
	if policy.DeletionTimestamp.IsZero() {
		if !containsString(policy.Finalizers, alertsPolicyDeleteFinalizer) {
			policy.Finalizers = append(policy.Finalizers, alertsPolicyDeleteFinalizer)
		}
	} else {
		return r.deleteAlertsPolicy(r.ctx, &policy, alertsPolicyDeleteFinalizer)
	}

	if policy.Spec.Equals(*policy.Status.AppliedSpec) {
		return ctrl.Result{}, nil
	}

	r.Log.Info("Reconciling", "policy", policy.Name)

	r.checkForExistingAlertsPolicy(&policy)

	if policy.Status.PolicyID != "" {
		err := r.updateAlertsPolicy(&policy)
		if err != nil {
			r.Log.Error(err, "error updating policy")
			return ctrl.Result{}, err
		}
	} else {
		err := r.createAlertsPolicy(&policy)
		if err != nil {
			r.Log.Error(err, "Error creating policy")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *AlertsPolicyReconciler) createAlertsPolicy(policy *nrv1.AlertsPolicy) error {
	defer r.txn.StartSegment("createAlertsPolicy").End()
	p := alerts.AlertsPolicyInput{}
	p.IncidentPreference = alerts.AlertsIncidentPreference(policy.Spec.IncidentPreference)
	p.Name = policy.Spec.Name

	r.Log.Info("Creating policy", "PolicyName", p.Name)
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
	defer r.txn.StartSegment("createConditions").End()
	r.Log.Info("creating conditions for policy")

	collectedErrors := new(customErrors.ErrorCollector)

	for i, condition := range policy.Spec.Conditions {
		var err error
		switch nrv1.GetAlertsConditionType(condition) {
		case "AlertsAPMCondition":
			err = r.createApmCondition(policy, &condition)
		case "AlertsNrqlCondition":
			err = r.createNrqlCondition(policy, &condition)
		}

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

type processedAlertsConditions struct {
	processed bool
	condition nrv1.AlertsPolicyCondition
}

func (r *AlertsPolicyReconciler) createOrUpdateCondition(policy *nrv1.AlertsPolicy, condition *nrv1.AlertsPolicyCondition) (*nrv1.AlertsPolicyCondition, error) {
	defer r.txn.StartSegment("createOrUpdateCondition").End()
	//loop through the policies, creating/updating as needed
	r.Log.Info("Checking on condition", "resourceName", condition.Name, "conditionName", condition.Spec.Name)
	//first we check to see if the name is set
	if condition.Name == "" {
		//If resource name is not set, let's see if a appliedSpec matches the NR condition name
		r.Log.Info("Condition name not set, checking name values")
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

			var err error
			switch nrv1.GetAlertsConditionType(*condition) {
			case "AlertsAPMCondition":
				err = r.createApmCondition(policy, condition)
			case "AlertsNrqlCondition":
				err = r.createNrqlCondition(policy, condition)
			}
			return condition, err
		}
	}

	var err error
	switch nrv1.GetAlertsConditionType(*condition) {
	case "AlertsAPMCondition":
		err = r.updateApmCondition(policy, condition)
	case "AlertsNrqlCondition":
		err = r.updateNrqlCondition(policy, condition)
	}

	return condition, err
}

func (r *AlertsPolicyReconciler) updateNrqlCondition(policy *nrv1.AlertsPolicy, condition *nrv1.AlertsPolicyCondition) error {
	defer r.txn.StartSegment("updateNrqlCondition").End()
	nrqlCondition := r.getAlertsNrqlConditionFromAlertsPolicyCondition(condition)

	r.Log.Info("Found nrql condition to update", "retrievedCondition", nrqlCondition)

	//Now check to confirm the NrqlCondition matches our PolicyCondition
	retrievedPolicyCondition := nrv1.AlertsPolicyCondition{}
	retrievedPolicyCondition.GenerateSpecFromNrqlConditionSpec(nrqlCondition.Spec)
	r.Log.Info("conditions", "retrieved", retrievedPolicyCondition, "condition", condition)

	if retrievedPolicyCondition.SpecHash() == condition.SpecHash() {
		r.Log.Info("existing NrqlCondition matches going to next")
		return nil
	}

	r.Log.Info("updating existing condition", "policyRegion", policy.Spec.Region, "policyId", policy.Status.PolicyID)

	nrqlCondition.Spec = condition.ReturnNrqlConditionSpec()
	//Set inherited values
	nrqlCondition.Spec.Region = policy.Spec.Region
	nrqlCondition.Spec.ExistingPolicyID = policy.Status.PolicyID
	nrqlCondition.Spec.APIKey = policy.Spec.APIKey
	nrqlCondition.Spec.APIKeySecret = policy.Spec.APIKeySecret
	nrqlCondition.Spec.AccountID = policy.Spec.AccountID

	err := r.Client.Update(r.ctx, &nrqlCondition)

	return err
}

func (r *AlertsPolicyReconciler) updateApmCondition(policy *nrv1.AlertsPolicy, condition *nrv1.AlertsPolicyCondition) error {
	defer r.txn.StartSegment("updateApmCondition").End()
	apmCondition := r.getApmConditionFromAlertsPolicyCondition(condition)

	r.Log.Info("Found apm condition to update", "retrievedCondition", apmCondition)

	//Now check to confirm the NrqlCondition matches our PolicyCondition
	retrievedPolicyCondition := nrv1.AlertsPolicyCondition{}
	retrievedPolicyCondition.GenerateSpecFromApmConditionSpec(apmCondition.Spec)
	r.Log.Info("conditions", "retrieved", retrievedPolicyCondition, "condition", condition)

	if retrievedPolicyCondition.SpecHash() == condition.SpecHash() {
		r.Log.Info("existing ApmCondition matches going to next")
		return nil
	}

	r.Log.Info("updating existing condition", "policyRegion", policy.Spec.Region, "policyId", policy.Status)

	apmCondition.Spec = condition.ReturnApmConditionSpec()
	//Set inherited values
	apmCondition.Spec.Region = policy.Spec.Region
	apmCondition.Spec.APIKey = policy.Spec.APIKey
	apmCondition.Spec.APIKeySecret = policy.Spec.APIKeySecret
	apmCondition.Spec.AccountID = policy.Spec.AccountID

	apmCondition.Spec.ExistingPolicyID = policy.Status.PolicyID

	r.Log.Info("updating existing condition", "alertsAPMCondition", apmCondition)

	err := r.Client.Update(r.ctx, &apmCondition)

	return err
}

func (r *AlertsPolicyReconciler) createOrUpdateConditions(policy *nrv1.AlertsPolicy) error {
	defer r.txn.StartSegment("createOrUpdateConditions").End()
	if reflect.DeepEqual(policy.Spec.Conditions, policy.Status.AppliedSpec.Conditions) {
		return nil
	}

	//build map of existing conditions so we can mark them off as processed and delete anything left over
	var existingConditions = make(map[string]processedAlertsConditions)
	for _, existing := range policy.Status.AppliedSpec.Conditions {
		existingConditions[existing.Name] = processedAlertsConditions{
			processed: false,
			condition: existing,
		}
	}

	collectedErrors := new(customErrors.ErrorCollector)

	for i, condition := range policy.Spec.Conditions {
		condition, err := r.createOrUpdateCondition(policy, &condition)
		if err != nil {
			r.Log.Error(err, "error creating condition")
			collectedErrors.Collect(err)
		}
		r.Log.Info("processed condition", "conditionName", condition.Name, "condition", condition)

		//add to the list of processed conditions
		existingConditions[condition.Name] = processedAlertsConditions{
			processed: true,
			condition: *condition,
		}

		//Now update the spec
		policy.Spec.Conditions[i] = *condition
		r.Log.Info("policy spec updated", "policy.Spec.Condition[i]", policy.Spec.Conditions[i])
	}

	r.Log.Info("now one last check for stragglers")
	for conditionName, processedCondition := range existingConditions {
		r.Log.Info("checking "+processedCondition.condition.Name, "bool is", processedCondition.processed)
		if !processedCondition.processed {
			r.Log.Info("Need to delete", "ppliedConditionName", conditionName)
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

func asOwner(p *nrv1.AlertsPolicy) metav1.OwnerReference {

	return metav1.OwnerReference{
		APIVersion: "v1", //this probablby shouldn't be hardcoded this way but doesn't work in tests for some reason
		Kind:       "AlertsPolicy",
		Name:       p.Name,
		UID:        p.UID,
		Controller: &trueVar,
	}
}

func (r *AlertsPolicyReconciler) createNrqlCondition(policy *nrv1.AlertsPolicy, condition *nrv1.AlertsPolicyCondition) error {
	defer r.txn.StartSegment("createNrqlCondition").End()
	var alertsNrqlCondition nrv1.AlertsNrqlCondition
	alertsNrqlCondition.GenerateName = policy.Name + "-condition-"
	alertsNrqlCondition.Namespace = policy.Namespace
	alertsNrqlCondition.Labels = policy.Labels
	alertsNrqlCondition.Spec = condition.ReturnNrqlConditionSpec()
	alertsNrqlCondition.Spec.Region = policy.Spec.Region
	alertsNrqlCondition.Spec.ExistingPolicyID = policy.Status.PolicyID
	alertsNrqlCondition.Spec.APIKey = policy.Spec.APIKey
	alertsNrqlCondition.Spec.APIKeySecret = policy.Spec.APIKeySecret
	alertsNrqlCondition.Spec.AccountID = policy.Spec.AccountID
	alertsNrqlCondition.Status.AppliedSpec = &nrv1.AlertsNrqlConditionSpec{}
	alertsNrqlCondition.OwnerReferences = append(alertsNrqlCondition.OwnerReferences, asOwner(policy))

	r.Log.Info("creating condition", "condition", condition.Name, "conditionName", condition.Spec.Name, "alertsNrqlCondition", alertsNrqlCondition)

	errCondition := r.Create(r.ctx, &alertsNrqlCondition)
	if errCondition != nil {
		r.Log.Error(errCondition, "error creating condition")
		return errCondition
	}

	condition.Name = alertsNrqlCondition.Name
	condition.Namespace = alertsNrqlCondition.Namespace

	r.Log.Info("created condition", "condition", condition.Name, "conditionName", condition.Spec.Name, "alertsNrqlCondition", alertsNrqlCondition)

	return nil
}

func (r *AlertsPolicyReconciler) createApmCondition(policy *nrv1.AlertsPolicy, condition *nrv1.AlertsPolicyCondition) error {
	defer r.txn.StartSegment("createApmCondition").End()
	var apmCondition nrv1.AlertsAPMCondition
	apmCondition.GenerateName = policy.Name + "-condition-"
	apmCondition.Namespace = policy.Namespace
	apmCondition.Labels = policy.Labels

	apmCondition.Spec = condition.ReturnApmConditionSpec()
	apmCondition.Spec.Region = policy.Spec.Region
	apmCondition.Spec.APIKey = policy.Spec.APIKey
	apmCondition.Spec.APIKeySecret = policy.Spec.APIKeySecret
	apmCondition.Spec.AccountID = policy.Spec.AccountID
	apmCondition.Status.AppliedSpec = &nrv1.AlertsAPMConditionSpec{}

	apmCondition.Spec.ExistingPolicyID = policy.Status.PolicyID
	apmCondition.OwnerReferences = append(apmCondition.OwnerReferences, asOwner(policy))

	r.Log.Info("creating apm condition", "condition", condition.Name, "conditionName", condition.Spec.Name, "alertsAPMCondition", apmCondition)
	errCondition := r.Create(r.ctx, &apmCondition)
	if errCondition != nil {
		r.Log.Error(errCondition, "error creating condition")
		return errCondition
	}

	condition.Name = apmCondition.Name
	condition.Namespace = apmCondition.Namespace

	r.Log.Info("created apm condition", "condition", condition.Name, "conditionName", condition.Spec.Name, "alertsAPMCondition", apmCondition, "actualCondition", condition.Spec)

	return nil
}

func (r *AlertsPolicyReconciler) deleteCondition(condition *nrv1.AlertsPolicyCondition) error {
	defer r.txn.StartSegment("deleteCondition").End()
	r.Log.Info("Deleting condition", "condition", condition.Name, "conditionName", condition.Spec.Name)

	var retrievedCondition runtime.Object
	switch nrv1.GetAlertsConditionType(*condition) {
	case "AlertsAPMCondition":
		returnedCondition := r.getApmConditionFromAlertsPolicyCondition(condition)
		retrievedCondition = &returnedCondition
	case "AlertsNrqlCondition":
		returnedCondition := r.getAlertsNrqlConditionFromAlertsPolicyCondition(condition)
		retrievedCondition = &returnedCondition
	}

	r.Log.Info("retrieved condition for deletion", "retrievedCondition", retrievedCondition)

	err := r.Delete(r.ctx, retrievedCondition)
	if err != nil {
		r.Log.Error(err, "error deleting condition resource")
		return err
	}

	return nil
}

func (r *AlertsPolicyReconciler) getAlertsNrqlConditionFromAlertsPolicyCondition(condition *nrv1.AlertsPolicyCondition) (nrqlCondition nrv1.AlertsNrqlCondition) {
	defer r.txn.StartSegment("getAlertsNrqlConditionFromAlertsPolicyCondition").End()
	r.Log.Info("condition before retrieval", "condition", condition)

	//throw away the error since empty conditions are expected
	_ = r.Client.Get(r.ctx, condition.GetNamespace(), &nrqlCondition)
	r.Log.Info("retrieved condition", "alertsNrqlCondition", nrqlCondition, "namespace", condition.GetNamespace())

	return
}

func (r *AlertsPolicyReconciler) getApmConditionFromAlertsPolicyCondition(condition *nrv1.AlertsPolicyCondition) (apmCondition nrv1.AlertsAPMCondition) {
	defer r.txn.StartSegment("getApmConditionFromAlertsPolicyCondition").End()
	r.Log.Info("apm condition before retrieval", "condition", condition)

	//throw away the error since empty conditions are expected
	_ = r.Client.Get(r.ctx, condition.GetNamespace(), &apmCondition)
	r.Log.Info("retrieved condition", "alertsAPMCondition", apmCondition, "namespace", condition.GetNamespace())

	return
}

func (r *AlertsPolicyReconciler) updateAlertsPolicy(policy *nrv1.AlertsPolicy) error {
	defer r.txn.StartSegment("updateAlertsPolicy").End()
	r.Log.Info("updating policy", "PolicyName", policy.Name)

	//only update policy if policy fields have changed
	updateInput := policy.Spec.ToAlertsPolicyUpdateInput()
	var updateResult *alerts.AlertsPolicy
	var err error

	if string(updateInput.IncidentPreference) != policy.Status.AppliedSpec.IncidentPreference || updateInput.Name != policy.Status.AppliedSpec.Name {
		r.Log.Info("need to update alert policy via New Relic API",
			"Alert AlertsPolicy Name", updateInput.Name,
			"incident preference ", policy.Status.AppliedSpec.IncidentPreference,
		)
		updateResult, err = r.Alerts.UpdatePolicyMutation(policy.Spec.AccountID, policy.Status.PolicyID, updateInput)
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
	defer r.txn.StartSegment("deleteAlertsPolicy").End()
	// The object is being deleted
	if containsString(policy.Finalizers, deleteFinalizer) {
		// catch invalid state
		if policy.Status.PolicyID == "" {
			r.Log.Info("No PolicyID set, removing finalizer")
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
				r.Log.Error(err, "Failed to delete Alert AlertsPolicy via New Relic API",
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
				r.Log.Error(err, "Failed to update k8s records for this policy after successfully deleting the policy via New Relic Alert API")
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
	defer r.txn.StartSegment("checkForExistingAlertsPolicy").End()
	if policy.Status.PolicyID != "" {
		return
	}

	r.Log.Info("checking for existing policy", "policy", policy.Name, "policyName", policy.Spec.Name)

	//if no policyId, get list of policies and compare name
	searchParams := alerts.AlertsPoliciesSearchCriteriaInput{}
	existingPolicies, err := r.Alerts.QueryPolicySearch(policy.Spec.AccountID, searchParams)

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

func (r *AlertsPolicyReconciler) deleteNewRelicAlertPolicy(policy *nrv1.AlertsPolicy) error {
	defer r.txn.StartSegment("deleteNewRelicAlertPolicy").End()
	r.Log.Info("Deleting policy", "policyName", policy.Spec.Name)

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
	defer r.txn.StartSegment("getAPIKeyOrSecret").End()
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
