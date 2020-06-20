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
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nralertsv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
)

// AlertsAPMConditionReconciler reconciles a AlertsAPMCondition object
type AlertsAPMConditionReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	AlertClientFunc func(string, string) (interfaces.NewRelicAlertsClient, error)
	apiKey          string
	Alerts          interfaces.NewRelicAlertsClient
}

// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=alertsapmconditions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=alertsapmconditions/status,verbs=get;update;patch

// nolint:gocyclo
func (r *AlertsAPMConditionReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("alertsapmcondition", req.NamespacedName)

	r.Log.Info("Starting reconcile action")
	var condition nralertsv1.AlertsAPMCondition
	err := r.Client.Get(ctx, req.NamespacedName, &condition)
	if err != nil {
		if strings.Contains(err.Error(), " not found") {
			r.Log.Info("Expected error 'not found' after condition deleted", "error", err)
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "Tried getting condition", "name", req.NamespacedName.String())
		return ctrl.Result{}, nil
	}

	r.apiKey = r.getAPIKeyOrSecret(condition)

	if r.apiKey == "" {
		return ctrl.Result{}, errors.New("api key is blank")
	}
	//initial alertsClient
	alertsClient, errAlertsClient := r.AlertClientFunc(r.apiKey, condition.Spec.Region)
	if errAlertsClient != nil {
		r.Log.Error(errAlertsClient, "Error thrown")
		return ctrl.Result{}, errAlertsClient
	}
	r.Alerts = alertsClient

	deleteFinalizer := "alertsapmconditions.finalizers.nr.k8s.newrelic.com"

	//examine DeletionTimestamp to determine if object is under deletion
	if condition.DeletionTimestamp.IsZero() {
		if !containsString(condition.Finalizers, deleteFinalizer) {
			condition.Finalizers = append(condition.Finalizers, deleteFinalizer)
		}
	} else {
		// The object is being deleted
		if containsString(condition.Finalizers, deleteFinalizer) {
			// catch invalid state
			if condition.Status.ConditionID == 0 {
				r.Log.Info("No Condition ID set, just removing finalizer")
				condition.Finalizers = removeString(condition.Finalizers, deleteFinalizer)
				if err := r.Client.Update(ctx, &condition); err != nil {
					r.Log.Error(err, "Failed to update condition after deleting New Relic Alert condition")
					return ctrl.Result{}, err
				}
			} else {
				// our finalizer is present, so lets handle any external dependency
				if err := r.deleteNewRelicAlertCondition(condition); err != nil {
					// if fail to delete the external dependency here, return with error
					// so that it can be retried
					r.Log.Error(err, "Failed to delete API Condition",
						"conditionId", condition.Status.ConditionID,
						"region", condition.Spec.Region,
						"Api Key", interfaces.PartialAPIKey(r.apiKey),
					)
					if err.Error() == "resource not found" {
						r.Log.Info("New Relic API returned resource not found, deleting condition resource")
					} else {
						r.Log.Error(err, "Failed to delete API Condition",
							"conditionId", condition.Status.ConditionID,
							"region", condition.Spec.Region,
							"Api Key", interfaces.PartialAPIKey(r.apiKey),
						)
						return ctrl.Result{}, err
					}
				}
				// remove our finalizer from the list and update it.
				r.Log.Info("New Relic Alert condition deleted, Removing finalizer")
				condition.Finalizers = removeString(condition.Finalizers, deleteFinalizer)
				if err := r.Client.Update(ctx, &condition); err != nil {
					r.Log.Error(err, "Failed to update condition after deleting New Relic Alert condition")
					return ctrl.Result{}, err
				}
			}
		}

		// Stop reconciliation as the item is being deleted
		r.Log.Info("All done with condition deletion", "conditionName", condition.Spec.Name)

		return ctrl.Result{}, nil
	}

	if reflect.DeepEqual(&condition.Spec, condition.Status.AppliedSpec) {
		return ctrl.Result{}, nil
	}

	r.Log.Info("Reconciling", "condition", condition.Name)

	//check if condition has condition id
	r.checkForExistingCondition(&condition)

	APICondition := condition.Spec.APICondition()

	if condition.Status.ConditionID != 0 && !reflect.DeepEqual(&condition.Spec, condition.Status.AppliedSpec) {
		r.Log.Info("updating condition", "ConditionName", condition.Name, "API fields", APICondition)
		APICondition.ID = condition.Status.ConditionID
		updatedCondition, err := alertsClient.UpdateCondition(APICondition)
		if err != nil {
			r.Log.Error(err, "failed to update condition")
		} else {
			condition.Status.AppliedSpec = &condition.Spec
			condition.Status.ConditionID = updatedCondition.ID
		}

		err = r.Client.Update(ctx, &condition)
		if err != nil {
			r.Log.Error(err, "tried updating condition status", "name", req.NamespacedName)
		}
	} else {
		r.Log.Info("Creating condition", "ConditionName", condition.Name, "API fields", APICondition)
		existingPolicyIDInt, err := strconv.Atoi(condition.Spec.ExistingPolicyID)
		if err != nil {
			r.Log.Error(err, "failed to read existing policy ID", "existingPolicyID", condition.Spec.ExistingPolicyID)
			return ctrl.Result{}, nil
		}

		createdCondition, err := alertsClient.CreateCondition(existingPolicyIDInt, APICondition)
		if err != nil {
			r.Log.Error(err, "failed to create condition",
				"conditionId", condition.Status.ConditionID,
				"region", condition.Spec.Region,
				"Api Key", interfaces.PartialAPIKey(r.apiKey),
			)
		} else {
			condition.Status.AppliedSpec = &condition.Spec
			condition.Status.ConditionID = createdCondition.ID
		}

		err = r.Client.Update(ctx, &condition)
		if err != nil {
			r.Log.Error(err, "tried updating condition status", "name", req.NamespacedName)
		}
	}

	return ctrl.Result{}, nil
}

func (r *AlertsAPMConditionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.AlertClientFunc = interfaces.InitializeAlertsClient

	return ctrl.NewControllerManagedBy(mgr).
		For(&nralertsv1.AlertsAPMCondition{}).
		Complete(r)
}

func (r *AlertsAPMConditionReconciler) checkForExistingCondition(condition *nralertsv1.AlertsAPMCondition) {
	if condition.Status.ConditionID == 0 {
		r.Log.Info("Checking for existing condition", "conditionName", condition.Name)

		//if no conditionId, get list of conditions and compare name
		existingPolicyIDInt, err := strconv.Atoi(condition.Spec.ExistingPolicyID)
		if err != nil {
			r.Log.Error(err, "failed to read existing policy ID", "existingPolicyID", condition.Spec.ExistingPolicyID)
			return
		}

		existingConditions, err := r.Alerts.ListConditions(existingPolicyIDInt)
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

func (r *AlertsAPMConditionReconciler) deleteNewRelicAlertCondition(condition nralertsv1.AlertsAPMCondition) error {
	r.Log.Info("Deleting condition", "conditionName", condition.Spec.Name)

	_, err := r.Alerts.DeleteCondition(condition.Status.ConditionID)
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

func (r *AlertsAPMConditionReconciler) getAPIKeyOrSecret(condition nralertsv1.AlertsAPMCondition) string {
	if condition.Spec.APIKey != "" {
		return condition.Spec.APIKey
	}

	if condition.Spec.APIKeySecret != (nralertsv1.NewRelicAPIKeySecret{}) {
		var apiKeySecret v1.Secret

		key := types.NamespacedName{Namespace: condition.Spec.APIKeySecret.Namespace, Name: condition.Spec.APIKeySecret.Name}

		if getErr := r.Client.Get(context.Background(), key, &apiKeySecret); getErr != nil {
			r.Log.Error(getErr, "Error retrieving secret", "secret", apiKeySecret)
			return ""
		}

		return string(apiKeySecret.Data[condition.Spec.APIKeySecret.KeyName])
	}

	return ""
}
