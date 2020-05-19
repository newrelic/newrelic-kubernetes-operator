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

	operatorAPI "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	customErrors "github.com/newrelic/newrelic-kubernetes-operator/errors"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
)

const (
	deleteFinalizer = "policies.finalizers.nr.k8s.newrelic.com"
)

// PolicyReconciler reconciles a Policy object
type PolicyReconciler struct {
	ClientFunc func(string, string) (interfaces.NewRelicAlertsClient, error)
	Alerts     interfaces.NewRelicAlertsClient
	Log        logr.Logger
	Scheme     *runtime.Scheme
	apiKey     string
	client.Client
	ctx context.Context
}

type processedConditions struct {
	processed bool
	condition operatorAPI.PolicyConditionSchema
}

// SetupWithManager creates a managed policy controller.
func (r *PolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorAPI.PolicySchema{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=policies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=policies/status,verbs=get;update;patch

// Reconcile performs the work to bring about the desired state of the policy.
func (r *PolicyReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	r.ctx = context.Background()
	_ = r.Log.WithValues("policy", req.NamespacedName)

	var policySchema operatorAPI.PolicySchema
	err := r.Client.Get(r.ctx, req.NamespacedName, &policySchema)
	if err != nil {
		// It is normal to Policy not found after deletion is normal
		if strings.Contains(err.Error(), " not found") {
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "Failed to get policy from cluster", "name", req.NamespacedName.String())
		return ctrl.Result{}, nil
	}

	apiKey, err := r.getAPIKey(policySchema)
	if err != nil {
		r.Log.Error(err, "Failed to get API key")
		return ctrl.Result{}, err
	}

	// Initialize the HTTP client
	client, err := r.ClientFunc(apiKey, policySchema.Spec.Region)
	if err != nil {
		r.Log.Error(err, "Failed to create HTTP client")
		return ctrl.Result{}, err
	}

	r.Alerts = client

	// Examine DeletionTimestamp to determine if object is under deletion
	if policySchema.DeletionTimestamp.IsZero() {
		if !containsString(policySchema.Finalizers, deleteFinalizer) {
			policySchema.Finalizers = append(policySchema.Finalizers, deleteFinalizer)
		}
	} else {
		return r.deletePolicy(r.ctx, &policySchema, deleteFinalizer)
	}

	// No-op if there is no work to do
	if policySchema.Spec.Equals(*policySchema.Status.AppliedSpec) {
		return ctrl.Result{}, nil
	}

	r.Log.Info("Reconciling",
		"policy.Spec.Conditions", policySchema.Spec.Conditions,
		"policy.Status.AppliedSpec.Conditions", policySchema.Status.AppliedSpec.Conditions)

	r.setExistingPolicyIDs(&policySchema)

	if policySchema.Status.PolicyID != 0 {
		err := r.updatePolicy(&policySchema)
		if err != nil {
			r.Log.Error(err, "Error updating policy")
			return ctrl.Result{}, err
		}
	} else {
		err := r.createPolicy(&policySchema)
		if err != nil {
			r.Log.Error(err, "Error creating policy")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *PolicyReconciler) createPolicy(policySchema *operatorAPI.PolicySchema) error {
	r.Log.Info("Creating policy", "name", policySchema.Name)

	created, err := r.Alerts.CreatePolicy(policySchema.Spec.ToPolicy())
	if err != nil {
		return err
	}

	policySchema.Status.PolicyID = created.ID

	if err = r.createConditions(policySchema); err != nil {
		return err
	}

	policySchema.Status.AppliedSpec = &policySchema.Spec

	if err = r.Client.Update(r.ctx, policySchema); err != nil {
		return err
	}

	return nil
}

func (r *PolicyReconciler) createConditions(policy *operatorAPI.PolicySchema) error {
	r.Log.Info("Creating conditions")

	collectedErrors := new(customErrors.ErrorCollector)
	for i, condition := range policy.Spec.Conditions {
		err := r.createCondition(policy, &condition)
		if err != nil {
			collectedErrors.Collect(err)
		} else {
			policy.Spec.Conditions[i] = condition
		}
	}

	if len(*collectedErrors) > 0 {
		return collectedErrors
	}

	return nil
}

func (r *PolicyReconciler) createOrUpdateCondition(policy *operatorAPI.PolicySchema, conditionSchema operatorAPI.PolicyConditionSchema) (*operatorAPI.PolicyConditionSchema, error) {
	r.Log.Info("Visiting condition", "resourceName", conditionSchema.Name, "conditionName", conditionSchema.Spec.Name)

	if conditionSchema.Name == "" {
		for _, appliedCondition := range policy.Status.AppliedSpec.Conditions {
			if appliedCondition.Spec.Name == conditionSchema.Spec.Name {
				conditionSchema.Namespace = appliedCondition.Namespace
				conditionSchema.Name = appliedCondition.Name
				break
			}
		}

		// Condition needs to be created
		if conditionSchema.Name == "" {
			if err := r.createCondition(policy, &conditionSchema); err != nil {
				return nil, err
			}

			return &conditionSchema, nil
		}
	}

	// Swallow error since this condition may not exist
	var nrqlConditionSchema operatorAPI.NrqlConditionSchema
	_ = r.Client.Get(r.ctx, conditionSchema.GetNamespace(), &nrqlConditionSchema)

	retrievedConditionSchema := operatorAPI.PolicyConditionSchema{Spec: nrqlConditionSchema.Spec}

	// No-op if there is no work to do
	if retrievedConditionSchema.SpecHash() == conditionSchema.SpecHash() {
		return &conditionSchema, nil
	}

	// Update the condition
	nrqlConditionSchema.Spec = conditionSchema.Spec
	nrqlConditionSchema.Spec.Region = policy.Spec.Region
	nrqlConditionSchema.Spec.ExistingPolicyID = policy.Status.PolicyID
	nrqlConditionSchema.Spec.APIKey = policy.Spec.APIKey
	nrqlConditionSchema.Spec.APIKeySecret = policy.Spec.APIKeySecret

	if err := r.Client.Update(r.ctx, &nrqlConditionSchema); err != nil {
		return nil, err
	}

	return &conditionSchema, nil
}

func (r *PolicyReconciler) createOrUpdateConditions(policy *operatorAPI.PolicySchema) error {
	// No-op if there is no work to do
	if reflect.DeepEqual(policy.Spec.Conditions, policy.Status.AppliedSpec.Conditions) {
		return nil
	}

	// Mark existing conditions as processed
	var existingConditions = make(map[string]processedConditions)
	for _, existing := range policy.Status.AppliedSpec.Conditions {
		existingConditions[existing.Name] = processedConditions{
			processed: false,
			condition: existing,
		}
	}

	errCollector := new(customErrors.ErrorCollector)
	for i, condition := range policy.Spec.Conditions {
		conditionSchema, err := r.createOrUpdateCondition(policy, condition)
		if err != nil {
			errCollector.Collect(err)
		}

		policy.Spec.Conditions[i] = *conditionSchema
		existingConditions[condition.Name] = processedConditions{
			processed: true,
			condition: condition,
		}
	}

	for conditionName, processedCondition := range existingConditions {
		r.Log.Info("checking "+processedCondition.condition.Name, "bool is", processedCondition.processed)
		if !processedCondition.processed {
			r.Log.Info("Need to delete", "ppliedConditionName", conditionName)
			err := r.deleteCondition(&processedCondition.condition)
			if err != nil {
				r.Log.Error(err, "error deleting condition resource")
				errCollector.Collect(err)
			}
		}

	}
	if len(*errCollector) > 0 {
		r.Log.Info("Errors encountered processing conditions", "collectedErrors", errCollector)
		return errCollector
	}

	r.Log.Info("all done", "policy.Spec", policy.Spec, "policy.Status.AppliedSpec.Conditions", policy.Status.AppliedSpec.Conditions)

	return nil
}

func (r *PolicyReconciler) createCondition(policy *operatorAPI.PolicySchema, conditionSchema *operatorAPI.PolicyConditionSchema) error {
	var condition operatorAPI.NrqlConditionSchema
	condition.GenerateName = policy.Name + "-condition-"
	condition.Namespace = policy.Namespace
	condition.Labels = policy.Labels
	condition.Spec = conditionSchema.Spec
	condition.Spec.Region = policy.Spec.Region
	condition.Spec.ExistingPolicyID = policy.Status.PolicyID
	condition.Spec.APIKey = policy.Spec.APIKey
	condition.Spec.APIKeySecret = policy.Spec.APIKeySecret
	condition.Status.AppliedSpec = &operatorAPI.NrqlConditionSpec{}

	r.Log.Info("Creating condition",
		"condition", conditionSchema.Name,
		"conditionName", conditionSchema.Spec.Name,
		"nrqlAlertCondition", condition)

	err := r.Create(r.ctx, &condition)
	if err != nil {
		return err
	}

	conditionSchema.Name = condition.Name
	conditionSchema.Namespace = condition.Namespace

	return nil
}

func (r *PolicyReconciler) deleteCondition(conditionSchema *operatorAPI.PolicyConditionSchema) error {
	r.Log.Info("Deleting condition", "condition", conditionSchema.Name, "conditionName", conditionSchema.Spec.Name)

	var retrievedCondition operatorAPI.NrqlConditionSchema
	r.Client.Get(r.ctx, conditionSchema.GetNamespace(), &retrievedCondition)
	if err := r.Delete(r.ctx, &retrievedCondition); err != nil {
		return err
	}

	return nil
}

func (r *PolicyReconciler) updatePolicy(policySchema *operatorAPI.PolicySchema) error {
	r.Log.Info("updating policy", "PolicyName", policySchema.Name)

	policy := policySchema.Spec.ToPolicy()
	policy.ID = policySchema.Status.PolicyID

	if policyRequiresUpdate(policy, policySchema) {
		updatedPolicy, err := r.Alerts.UpdatePolicy(policy)
		if err != nil {
			return err
		}

		policySchema.Status.PolicyID = updatedPolicy.ID
	}

	errConditions := r.createOrUpdateConditions(policySchema)
	if errConditions != nil {
		return errConditions
	}

	policySchema.Status.AppliedSpec = &policySchema.Spec

	if err := r.Client.Update(r.ctx, policySchema); err != nil {
		return err
	}

	return nil
}

func policyRequiresUpdate(policy alerts.Policy, policySchema *operatorAPI.PolicySchema) bool {
	return string(policy.IncidentPreference) != policySchema.Status.AppliedSpec.IncidentPreference ||
		policy.Name != policySchema.Status.AppliedSpec.Name
}

func (r *PolicyReconciler) deletePolicy(ctx context.Context, policy *operatorAPI.PolicySchema, deleteFinalizer string) (ctrl.Result, error) {
	if !containsString(policy.Finalizers, deleteFinalizer) {
		return ctrl.Result{}, nil
	}

	r.Log.Info("Deleting policy", "policyName", policy.Spec.Name)

	if policy.Status.PolicyID == 0 {
		policy.Finalizers = removeStringFromSlice(policy.Finalizers, deleteFinalizer)
	} else {
		collectedErrors := new(customErrors.ErrorCollector)
		for _, condition := range policy.Status.AppliedSpec.Conditions {
			err := r.deleteCondition(&condition)
			if err != nil {
				collectedErrors.Collect(err)
			}
		}

		if len(*collectedErrors) > 0 {
			return ctrl.Result{}, collectedErrors
		}

		_, err := r.Alerts.DeletePolicy(policy.Status.PolicyID)
		if err != nil {
			return ctrl.Result{}, err
		}

		policy.Finalizers = removeStringFromSlice(policy.Finalizers, deleteFinalizer)
		if err := r.Client.Update(ctx, policy); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *PolicyReconciler) setExistingPolicyIDs(policy *operatorAPI.PolicySchema) {
	if policy.Status.PolicyID != 0 {
		return
	}

	r.Log.Info("Checking for existing policy", "policy", policy.Name, "policyName", policy.Spec.Name)
	params := &alerts.ListPoliciesParams{
		Name: policy.Spec.Name,
	}
	policies, _ := r.Alerts.ListPolicies(params)
	for _, existingPolicy := range policies {
		if existingPolicy.Name == policy.Spec.Name {
			policy.Status.PolicyID = existingPolicy.ID
			break
		}
	}
}

func (r *PolicyReconciler) getAPIKey(policy operatorAPI.PolicySchema) (string, error) {
	if policy.Spec.APIKey != "" {
		return policy.Spec.APIKey, nil
	}

	if policy.Spec.APIKeySecret != (operatorAPI.NewRelicAPIKeySecret{}) {
		key := types.NamespacedName{Namespace: policy.Spec.APIKeySecret.Namespace, Name: policy.Spec.APIKeySecret.Name}

		var secret v1.Secret
		err := r.Client.Get(context.Background(), key, &secret)
		if err != nil {
			return "", err
		}

		return string(secret.Data[policy.Spec.APIKeySecret.KeyName]), nil
	}

	return "", errors.New("No API key found")
}
