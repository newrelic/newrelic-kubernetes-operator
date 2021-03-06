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

	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	v1 "k8s.io/api/core/v1"
	kErr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	customErrors "github.com/newrelic/newrelic-kubernetes-operator/errors"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
)

// PolicyReconciler reconciles a Policy object
type PolicyReconciler struct {
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

// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=policies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=policies/status,verbs=get;update;patch

func (r *PolicyReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	r.ctx = context.Background()
	_ = r.Log.WithValues("policy", req.NamespacedName)

	r.txn = r.NewRelicAgent.StartTransaction("Reconcile/Policy")
	defer r.txn.End()

	var policy nrv1.Policy
	err := r.Client.Get(r.ctx, req.NamespacedName, &policy)
	if err != nil {
		if kErr.IsNotFound(err) {
			r.Log.Info("Policy 'not found' after being deleted. This is expected and no cause for alarm", "error", err)
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "Failed to GET policy", "name", req.NamespacedName.String())
		return ctrl.Result{}, err
	}
	r.Log.Info("Starting reconcile action")
	r.Log.Info("policy", "policy.Spec.Condition", policy.Spec.Conditions, "policy.status.applied.conditions", policy.Status.AppliedSpec.Conditions)

	r.apiKey, err = r.getAPIKeyOrSecret(policy)
	if err != nil {
		return ctrl.Result{}, err
	}

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

	deleteFinalizer := "policies.finalizers.nr.k8s.newrelic.com"

	//examine DeletionTimestamp to determine if object is under deletion
	if policy.DeletionTimestamp.IsZero() {
		if !containsString(policy.Finalizers, deleteFinalizer) {
			policy.Finalizers = append(policy.Finalizers, deleteFinalizer)
		}
	} else {
		return r.deletePolicy(r.ctx, &policy, deleteFinalizer)
	}

	if policy.Spec.Equals(*policy.Status.AppliedSpec) {
		return ctrl.Result{}, nil
	}

	r.Log.Info("Reconciling", "policy", policy.Name)

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
			r.Log.Error(err, "Error creating policy")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *PolicyReconciler) createPolicy(policy *nrv1.Policy) error {
	defer r.txn.StartSegment("createPolicy").End()
	r.Log.Info("Creating policy", "PolicyName", policy.Name)
	APIPolicy := policy.Spec.APIPolicy()
	createdPolicy, err := r.Alerts.CreatePolicy(APIPolicy)
	if err != nil {
		r.Log.Error(err, "failed to create policy via New Relic API",
			"policyId", policy.Status.PolicyID,
			"region", policy.Spec.Region,
			"Api Key", interfaces.PartialAPIKey(r.apiKey),
		)

		return err
	}
	policy.Status.PolicyID = createdPolicy.ID

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

func (r *PolicyReconciler) createConditions(policy *nrv1.Policy) error {
	defer r.txn.StartSegment("createConditions").End()
	r.Log.Info("initial policy creation so create all policies")
	collectedErrors := new(customErrors.ErrorCollector)
	for i, condition := range policy.Spec.Conditions {

		var err error
		switch nrv1.GetConditionType(condition) {
		case "ApmAlertCondition":
			err = r.createApmCondition(policy, &condition)
		case "NrqlAlertCondition":
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

type processedConditions struct {
	processed bool
	condition nrv1.PolicyCondition
}

func (r *PolicyReconciler) createOrUpdateCondition(policy *nrv1.Policy, condition *nrv1.PolicyCondition) (*nrv1.PolicyCondition, error) {
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
			switch nrv1.GetConditionType(*condition) {
			case "ApmAlertCondition":
				err = r.createApmCondition(policy, condition)
			case "NrqlAlertCondition":
				err = r.createNrqlCondition(policy, condition)
			}
			return condition, err
		}
	}

	var err error
	switch nrv1.GetConditionType(*condition) {
	case "ApmAlertCondition":
		err = r.updateApmCondition(policy, condition)
	case "NrqlAlertCondition":
		err = r.updateNrqlCondition(policy, condition)
	}

	return condition, err
}

func (r *PolicyReconciler) updateNrqlCondition(policy *nrv1.Policy, condition *nrv1.PolicyCondition) error {
	defer r.txn.StartSegment("updateNrqlCondition").End()
	nrqlAlertCondition := r.getNrqlConditionFromPolicyCondition(condition)

	r.Log.Info("Found nrql condition to update", "retrievedCondition", nrqlAlertCondition)

	//Now check to confirm the NrqlCondition matches our PolicyCondition
	retrievedPolicyCondition := nrv1.PolicyCondition{}
	retrievedPolicyCondition.GenerateSpecFromNrqlConditionSpec(nrqlAlertCondition.Spec)
	r.Log.Info("conditions", "retrieved", retrievedPolicyCondition, "condition", condition)

	if retrievedPolicyCondition.SpecHash() == condition.SpecHash() {
		r.Log.Info("existing NrqlCondition matches going to next")

		return nil
	}

	r.Log.Info("updating existing condition", "policyRegion", policy.Spec.Region, "policyId", policy.Status.PolicyID)

	nrqlAlertCondition.Spec = condition.ReturnNrqlConditionSpec()
	//Set inherited values
	nrqlAlertCondition.Spec.Region = policy.Spec.Region
	nrqlAlertCondition.Spec.ExistingPolicyID = policy.Status.PolicyID
	nrqlAlertCondition.Spec.APIKey = policy.Spec.APIKey
	nrqlAlertCondition.Spec.APIKeySecret = policy.Spec.APIKeySecret

	return r.Client.Update(r.ctx, &nrqlAlertCondition)
}

func (r *PolicyReconciler) updateApmCondition(policy *nrv1.Policy, condition *nrv1.PolicyCondition) error {
	defer r.txn.StartSegment("updateApmCondition").End()
	apmAlertCondition := r.getApmConditionFromPolicyCondition(condition)

	r.Log.Info("Found apm condition to update", "retrievedCondition", apmAlertCondition)

	//Now check to confirm the NrqlCondition matches our PolicyCondition
	retrievedPolicyCondition := nrv1.PolicyCondition{}
	retrievedPolicyCondition.GenerateSpecFromApmConditionSpec(apmAlertCondition.Spec)
	r.Log.Info("conditions", "retrieved", retrievedPolicyCondition, "condition", condition)

	if retrievedPolicyCondition.SpecHash() == condition.SpecHash() {
		r.Log.Info("existing ApmCondition matches going to next")
		return nil
	}

	r.Log.Info("updating existing condition", "policyRegion", policy.Spec.Region, "policyId", policy.Status)

	apmAlertCondition.Spec = condition.ReturnApmConditionSpec()
	//Set inherited values
	apmAlertCondition.Spec.Region = policy.Spec.Region
	apmAlertCondition.Spec.ExistingPolicyID = policy.Status.PolicyID
	apmAlertCondition.Spec.APIKey = policy.Spec.APIKey
	apmAlertCondition.Spec.APIKeySecret = policy.Spec.APIKeySecret

	r.Log.Info("updating existing condition", "apmAlertCondition", apmAlertCondition)

	return r.Client.Update(r.ctx, &apmAlertCondition)
}

func (r *PolicyReconciler) createOrUpdateConditions(policy *nrv1.Policy) error {
	defer r.txn.StartSegment("createOrUpdateConditions").End()
	if reflect.DeepEqual(policy.Spec.Conditions, policy.Status.AppliedSpec.Conditions) {
		return nil
	}

	//build map of existing conditions so we can mark them off as processed and delete anything left over
	var existingConditions = make(map[string]processedConditions)
	for _, existing := range policy.Status.AppliedSpec.Conditions {
		existingConditions[existing.Name] = processedConditions{
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
		existingConditions[condition.Name] = processedConditions{
			processed: true,
			condition: *condition,
		}

		//Now update the spec
		policy.Spec.Conditions[i] = *condition
		r.Log.Info("policy spec updated", "policy.Spec.Condition[i]", policy.Spec.Conditions[i])
	}

	r.Log.Info("now one last check for stragglers")
	//now we check for any left behind conditions that weren't processed

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

func (r *PolicyReconciler) createNrqlCondition(policy *nrv1.Policy, condition *nrv1.PolicyCondition) error {
	defer r.txn.StartSegment("createNrqlCondition").End()
	var nrqlAlertCondition nrv1.NrqlAlertCondition
	nrqlAlertCondition.GenerateName = policy.Name + "-condition-"
	nrqlAlertCondition.Namespace = policy.Namespace
	nrqlAlertCondition.Labels = policy.Labels
	//TODO: no clue if this is needed, I'm guessing no
	//condition.OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(policy, conditionKind)},

	nrqlAlertCondition.Spec = condition.ReturnNrqlConditionSpec()
	nrqlAlertCondition.Spec.Region = policy.Spec.Region
	nrqlAlertCondition.Spec.ExistingPolicyID = policy.Status.PolicyID
	nrqlAlertCondition.Spec.APIKey = policy.Spec.APIKey
	nrqlAlertCondition.Spec.APIKeySecret = policy.Spec.APIKeySecret
	nrqlAlertCondition.Status.AppliedSpec = &nrv1.NrqlAlertConditionSpec{}

	r.Log.Info("creating nrql condition", "condition", condition.Name, "conditionName", condition.Spec.Name, "nrqlAlertCondition", nrqlAlertCondition)
	errCondition := r.Create(r.ctx, &nrqlAlertCondition)
	if errCondition != nil {
		r.Log.Error(errCondition, "error creating condition")
		return errCondition
	}
	condition.Name = nrqlAlertCondition.Name //created from generated name
	condition.Namespace = nrqlAlertCondition.Namespace

	r.Log.Info("created condition", "condition", condition.Name, "conditionName", condition.Spec.Name, "nrqlAlertCondition", nrqlAlertCondition, "actualCondition", condition.Spec)

	return nil
}

func (r *PolicyReconciler) createApmCondition(policy *nrv1.Policy, condition *nrv1.PolicyCondition) error {
	defer r.txn.StartSegment("createApmCondition").End()
	var apmAlertCondition nrv1.ApmAlertCondition
	apmAlertCondition.GenerateName = policy.Name + "-condition-"
	apmAlertCondition.Namespace = policy.Namespace
	apmAlertCondition.Labels = policy.Labels
	//TODO: no clue if this is needed, I'm guessing no
	//condition.OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(policy, conditionKind)},

	apmAlertCondition.Spec = condition.ReturnApmConditionSpec()
	apmAlertCondition.Spec.Region = policy.Spec.Region
	apmAlertCondition.Spec.ExistingPolicyID = policy.Status.PolicyID
	apmAlertCondition.Spec.APIKey = policy.Spec.APIKey
	apmAlertCondition.Spec.APIKeySecret = policy.Spec.APIKeySecret
	apmAlertCondition.Status.AppliedSpec = &nrv1.ApmAlertConditionSpec{}

	r.Log.Info("creating apm condition", "condition", condition.Name, "conditionName", condition.Spec.Name, "apmAlertCondition", apmAlertCondition)
	errCondition := r.Create(r.ctx, &apmAlertCondition)
	if errCondition != nil {
		r.Log.Error(errCondition, "error creating condition")
		return errCondition
	}
	condition.Name = apmAlertCondition.Name //created from generated name
	condition.Namespace = apmAlertCondition.Namespace

	r.Log.Info("created apm condition", "condition", condition.Name, "conditionName", condition.Spec.Name, "apmAlertCondition", apmAlertCondition, "actualCondition", condition.Spec)

	return nil
}

func (r *PolicyReconciler) deleteCondition(condition *nrv1.PolicyCondition) error {
	defer r.txn.StartSegment("deleteCondition").End()
	r.Log.Info("Deleting condition", "condition", condition.Name, "conditionName", condition.Spec.Name)

	var retrievedCondition runtime.Object
	switch nrv1.GetConditionType(*condition) {
	case "ApmAlertCondition":
		returnedCondition := r.getApmConditionFromPolicyCondition(condition)
		retrievedCondition = &returnedCondition
	case "NrqlAlertCondition":
		returnedCondition := r.getNrqlConditionFromPolicyCondition(condition)
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

func (r *PolicyReconciler) getNrqlConditionFromPolicyCondition(condition *nrv1.PolicyCondition) (nrqlAlertCondition nrv1.NrqlAlertCondition) {
	defer r.txn.StartSegment("getNrqlConditionFromPolicyCondition").End()
	r.Log.Info("nrql condition before retrieval", "condition", condition)
	//throw away the error since empty conditions are expected
	_ = r.Client.Get(r.ctx, condition.GetNamespace(), &nrqlAlertCondition)
	r.Log.Info("retrieved condition", "nrqlAlertCondition", nrqlAlertCondition, "namespace", condition.GetNamespace())

	return
}

func (r *PolicyReconciler) getApmConditionFromPolicyCondition(condition *nrv1.PolicyCondition) (apmAlertCondition nrv1.ApmAlertCondition) {
	defer r.txn.StartSegment("getApmConditionFromPolicyCondition").End()
	r.Log.Info("apm condition before retrieval", "condition", condition)
	//throw away the error since empty conditions are expected
	_ = r.Client.Get(r.ctx, condition.GetNamespace(), &apmAlertCondition)
	r.Log.Info("retrieved condition", "apmAlertCondition", apmAlertCondition, "namespace", condition.GetNamespace())

	return
}

func (r *PolicyReconciler) updatePolicy(policy *nrv1.Policy) error {
	defer r.txn.StartSegment("updatePolicy").End()
	r.Log.Info("updating policy", "PolicyName", policy.Name)

	//only update policy if policy fields have changed
	APIPolicy := policy.Spec.APIPolicy()
	APIPolicy.ID = policy.Status.PolicyID
	var updatedPolicy *alerts.Policy
	var err error

	if string(APIPolicy.IncidentPreference) != policy.Status.AppliedSpec.IncidentPreference || APIPolicy.Name != policy.Status.AppliedSpec.Name {
		r.Log.Info("need to update alert policy via New Relic API",
			"Alert Policy Name", APIPolicy.Name,
			"incident preference ", policy.Status.AppliedSpec.IncidentPreference,
		)
		updatedPolicy, err = r.Alerts.UpdatePolicy(APIPolicy)
		if err != nil {
			r.Log.Error(err, "failed to update policy via New Relic API",
				"policyId", policy.Status.PolicyID,
				"region", policy.Spec.Region,
				"Api Key", interfaces.PartialAPIKey(r.apiKey),
			)

			return err
		}
		policy.Status.PolicyID = updatedPolicy.ID
	}

	errConditions := r.createOrUpdateConditions(policy)
	if errConditions != nil {
		r.Log.Error(errConditions, "error creating or updating conditions")

		return errConditions
	}
	r.Log.Info("policySpecx before update", "policy.Spec", policy.Spec)

	policy.Status.AppliedSpec = &policy.Spec

	err = r.Client.Update(r.ctx, policy)
	if err != nil {
		r.Log.Error(err, "failed to update policy status", "name", policy.Name)

		return err
	}

	return nil
}

func (r *PolicyReconciler) deletePolicy(ctx context.Context, policy *nrv1.Policy, deleteFinalizer string) (ctrl.Result, error) {
	defer r.txn.StartSegment("deletePolicy").End()
	// The object is being deleted
	if containsString(policy.Finalizers, deleteFinalizer) {
		// catch invalid state
		if policy.Status.PolicyID == 0 {
			r.Log.Info("No Condition ID set, just removing finalizer")
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
				r.Log.Error(err, "Failed to delete Alert Policy via New Relic API",
					"policyId", policy.Status.PolicyID,
					"region", policy.Spec.Region,
					"Api Key", interfaces.PartialAPIKey(r.apiKey),
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

func (r *PolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nrv1.Policy{}).
		Complete(r)
}

func (r *PolicyReconciler) checkForExistingPolicy(policy *nrv1.Policy) {
	defer r.txn.StartSegment("checkForExistingPolicy").End()
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
				"Api Key", interfaces.PartialAPIKey(r.apiKey),
			)
		} else {
			for _, existingPolicy := range existingPolicies {
				if existingPolicy.Name == policy.Spec.Name {
					r.Log.Info("Matched on existing policy, updating PolicyId", "policyId", existingPolicy.ID)
					policy.Status.PolicyID = existingPolicy.ID
					break
				}
			}
		}
	}
}

func (r *PolicyReconciler) deleteNewRelicAlertPolicy(policy *nrv1.Policy) error {
	defer r.txn.StartSegment("deleteNewRelicAlertPolicy").End()
	r.Log.Info("Deleting policy", "policyName", policy.Spec.Name)
	_, err := r.Alerts.DeletePolicy(policy.Status.PolicyID)
	if err != nil {
		r.Log.Error(err, "Error deleting policy via New Relic API",
			"policyId", policy.Status.PolicyID,
			"region", policy.Spec.Region,
			"Api Key", interfaces.PartialAPIKey(r.apiKey),
		)

		return err
	}

	return nil
}

func (r *PolicyReconciler) getAPIKeyOrSecret(policy nrv1.Policy) (string, error) {
	defer r.txn.StartSegment("getAPIKeyOrSecret").End()
	if policy.Spec.APIKey != "" {
		return policy.Spec.APIKey, nil
	}

	if policy.Spec.APIKeySecret != (nrv1.NewRelicAPIKeySecret{}) {
		key := types.NamespacedName{Namespace: policy.Spec.APIKeySecret.Namespace, Name: policy.Spec.APIKeySecret.Name}
		var apiKeySecret v1.Secret
		getErr := r.Client.Get(context.Background(), key, &apiKeySecret)
		if getErr != nil {
			r.Log.Error(getErr, "Failed to retrieve secret", "secret", apiKeySecret)
			return "", getErr
		}

		return string(apiKeySecret.Data[policy.Spec.APIKeySecret.KeyName]), nil
	}

	return "", nil
}
