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

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorAPI "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
)

// NrqlConditionReconciler reconciles a NrqlCondition object
type NrqlConditionReconciler struct {
	client.Client
	Alerts          interfaces.NewRelicAlertsClient
	Log             logr.Logger
	Scheme          *runtime.Scheme
	AlertClientFunc func(string, string) (interfaces.NewRelicAlertsClient, error)
	apiKey          string
}

// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=nrqlalertconditions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=nrqlalertconditions/status,verbs=get;update;patch

func (r *NrqlConditionReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("nrqlalertcondition", req.NamespacedName)

	r.Log.Info("Starting reconcile action")
	var conditionSchema operatorAPI.NrqlConditionSchema
	err := r.Client.Get(ctx, req.NamespacedName, &conditionSchema)
	if err != nil {
		if strings.Contains(err.Error(), " not found") {
			r.Log.Info("Expected error 'not found' after condition deleted", "error", err)
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "Tried getting condition", "name", req.NamespacedName.String())
		return ctrl.Result{}, nil
	}

	r.apiKey = r.getAPIKeyOrSecret(conditionSchema)

	if r.apiKey == "" {
		return ctrl.Result{}, errors.New("api key is blank")
	}
	//initial alertsClient
	alertsClient, errAlertsClient := r.AlertClientFunc(r.apiKey, conditionSchema.Spec.Region)
	if errAlertsClient != nil {
		r.Log.Error(errAlertsClient, "Error thrown")
		return ctrl.Result{}, errAlertsClient
	}
	r.Alerts = alertsClient

	deleteFinalizer := "nrqlalertconditions.finalizers.nr.k8s.newrelic.com"

	//examine DeletionTimestamp to determine if object is under deletion
	if conditionSchema.DeletionTimestamp.IsZero() {
		if !containsString(conditionSchema.Finalizers, deleteFinalizer) {
			conditionSchema.Finalizers = append(conditionSchema.Finalizers, deleteFinalizer)
		}
	} else {
		// The object is being deleted
		if containsString(conditionSchema.Finalizers, deleteFinalizer) {
			// catch invalid state
			if conditionSchema.Status.ConditionID == 0 {
				r.Log.Info("No Condition ID set, just removing finalizer")
				conditionSchema.Finalizers = removeStringFromSlice(conditionSchema.Finalizers, deleteFinalizer)
				if err := r.Client.Update(ctx, &conditionSchema); err != nil {
					r.Log.Error(err, "Failed to update condition after deleting New Relic Alert condition")
					return ctrl.Result{}, err
				}
			} else {
				// our finalizer is present, so lets handle any external dependency
				if err := r.deleteNewRelicAlertCondition(conditionSchema); err != nil {
					// if fail to delete the external dependency here, return with error
					// so that it can be retried
					r.Log.Error(err, "Failed to delete API Condition",
						"conditionId", conditionSchema.Status.ConditionID,
						"region", conditionSchema.Spec.Region,
						"Api Key", interfaces.PartialAPIKey(r.apiKey),
					)
					if err.Error() == "resource not found" {
						r.Log.Info("New Relic API returned resource not found, deleting condition resource")
					} else {
						r.Log.Error(err, "Failed to delete API Condition",
							"conditionId", conditionSchema.Status.ConditionID,
							"region", conditionSchema.Spec.Region,
							"Api Key", interfaces.PartialAPIKey(r.apiKey),
						)
						return ctrl.Result{}, err
					}
				}
				// remove our finalizer from the list and update it.
				r.Log.Info("New Relic Alert condition deleted, Removing finalizer")
				conditionSchema.Finalizers = removeStringFromSlice(conditionSchema.Finalizers, deleteFinalizer)
				if err := r.Client.Update(ctx, &conditionSchema); err != nil {
					r.Log.Error(err, "Failed to update condition after deleting New Relic Alert condition")
					return ctrl.Result{}, err
				}
			}
		}

		// Stop reconciliation as the item is being deleted
		r.Log.Info("All done with condition deletion", "conditionName", conditionSchema.Spec.Name)

		return ctrl.Result{}, nil
	}

	if reflect.DeepEqual(&conditionSchema.Spec, conditionSchema.Status.AppliedSpec) {
		return ctrl.Result{}, nil
	}

	r.Log.Info("Reconciling", "condition", conditionSchema.Name)

	//check if condition has condition id
	r.checkForExistingCondition(&conditionSchema)

	condition := conditionSchema.Spec.ToCondition()

	if conditionSchema.Status.ConditionID != 0 && !reflect.DeepEqual(&conditionSchema.Spec, conditionSchema.Status.AppliedSpec) {
		r.Log.Info("updating condition", "ConditionName", conditionSchema.Name, "API fields", condition)
		condition.ID = conditionSchema.Status.ConditionID
		updatedCondition, err := alertsClient.UpdateNrqlCondition(condition)
		if err != nil {
			r.Log.Error(err, "failed to update condition")
		} else {
			conditionSchema.Status.AppliedSpec = &conditionSchema.Spec
			conditionSchema.Status.ConditionID = updatedCondition.ID
		}

		err = r.Client.Update(ctx, &conditionSchema)
		if err != nil {
			r.Log.Error(err, "tried updating condition status", "name", req.NamespacedName)
		}
	} else {
		r.Log.Info("Creating condition", "ConditionName", conditionSchema.Name, "API fields", condition)
		createdCondition, err := alertsClient.CreateNrqlCondition(conditionSchema.Spec.ExistingPolicyID, condition)
		if err != nil {
			r.Log.Error(err, "failed to create condition",
				"conditionId", conditionSchema.Status.ConditionID,
				"region", conditionSchema.Spec.Region,
				"Api Key", interfaces.PartialAPIKey(r.apiKey),
			)
		} else {
			conditionSchema.Status.AppliedSpec = &conditionSchema.Spec
			conditionSchema.Status.ConditionID = createdCondition.ID
		}

		err = r.Client.Update(ctx, &conditionSchema)
		if err != nil {
			r.Log.Error(err, "tried updating condition status", "name", req.NamespacedName)
		}
	}

	return ctrl.Result{}, nil
}

func (r *NrqlConditionReconciler) checkForExistingCondition(condition *operatorAPI.NrqlConditionSchema) {
	if condition.Status.ConditionID == 0 {
		r.Log.Info("Checking for existing condition", "conditionName", condition.Name)
		//if no conditionId, get list of conditions and compare name
		existingConditions, err := r.Alerts.ListNrqlConditions(condition.Spec.ExistingPolicyID)
		if err != nil {
			r.Log.Error(err, "failed to get list of NRQL conditions from New Relic API",
				"conditionId", condition.Status.ConditionID,
				"region", condition.Spec.Region,
				"Api Key", interfaces.PartialAPIKey(r.apiKey),
			)
		} else {
			for _, existingCondition := range existingConditions {
				if existingCondition.Name == condition.Spec.Name {
					r.Log.Info("Matched on existing condition, updating ConditionId", "conditionId", existingCondition.ID)
					condition.Status.ConditionID = existingCondition.ID
					break
				}
			}
		}
	}
}

func (r *NrqlConditionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.AlertClientFunc = interfaces.InitializeAlertsClient
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorAPI.NrqlConditionSchema{}).
		Complete(r)
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func (r *NrqlConditionReconciler) deleteNewRelicAlertCondition(condition operatorAPI.NrqlConditionSchema) error {
	r.Log.Info("Deleting condition", "conditionName", condition.Spec.Name)
	_, err := r.Alerts.DeleteNrqlCondition(condition.Status.ConditionID)
	if err != nil {
		r.Log.Error(err, "Error deleting condition",
			"conditionId", condition.Status.ConditionID,
			"region", condition.Spec.Region,
			"Api Key", interfaces.PartialAPIKey(r.apiKey),
		)
		return err
	}
	return nil
}

func removeStringFromSlice(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func (r *NrqlConditionReconciler) getAPIKeyOrSecret(condition operatorAPI.NrqlConditionSchema) string {

	if condition.Spec.APIKey != "" {
		return condition.Spec.APIKey
	}
	if condition.Spec.APIKeySecret != (operatorAPI.NewRelicAPIKeySecret{}) {
		key := types.NamespacedName{Namespace: condition.Spec.APIKeySecret.Namespace, Name: condition.Spec.APIKeySecret.Name}
		var apiKeySecret v1.Secret
		if getErr := r.Client.Get(context.Background(), key, &apiKeySecret); getErr != nil {
			r.Log.Error(getErr, "Error retrieving secret", "secret", apiKeySecret)
			return ""
		}
		return string(apiKeySecret.Data[condition.Spec.APIKeySecret.KeyName])
	}
	return ""
}
