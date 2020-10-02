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

	v1 "k8s.io/api/core/v1"
	kErr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/newrelic-client-go/pkg/alerts"

	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
)

const (
	alertsNrqlConditionDeleteFinalizer = "alertsnrqlconditions.finalizers.nr.k8s.newrelic.com"
)

// AlertsNrqlConditionReconciler reconciles a AlertsNrqlCondition object
type AlertsNrqlConditionReconciler struct {
	client.Client
	Alerts          interfaces.NewRelicAlertsClient
	Log             logr.Logger
	Scheme          *runtime.Scheme
	AlertClientFunc func(string, string) (interfaces.NewRelicAlertsClient, error)
	apiKey          string
	NewRelicAgent   newrelic.Application
	txn             *newrelic.Transaction
}

// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=alertsnrqlconditions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=alertsnrqlconditions/status,verbs=get;update;patch

// Reconcile is responsible for reconciling the spec and state of the AlertsNrqlCondition.
func (r *AlertsNrqlConditionReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) { //nolint: gocyclo
	ctx := context.Background()
	_ = r.Log.WithValues("alertsnrqlcondition", req.NamespacedName)
	r.txn = r.NewRelicAgent.StartTransaction("Reconcile/Alerts/NrqlCondition")
	defer r.txn.End()

	r.Log.Info("starting reconcile action")
	var condition nrv1.AlertsNrqlCondition

	err := r.Client.Get(ctx, req.NamespacedName, &condition)
	if err != nil {
		if kErr.IsNotFound(err) {
			r.Log.Info("Expected error 'not found' after condition deleted", "error", err)
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "Tried getting condition", "name", req.NamespacedName.String())
		return ctrl.Result{}, err
	}

	r.apiKey, err = r.getAPIKeyOrSecret(condition)
	if err != nil {
		return ctrl.Result{}, err
	}

	if r.apiKey == "" {
		return ctrl.Result{}, errors.New("api key is blank")
	}

	alertsClient, errAlertsClient := r.AlertClientFunc(r.apiKey, condition.Spec.Region)
	if errAlertsClient != nil {
		r.Log.Error(errAlertsClient, "Error thrown")
		return ctrl.Result{}, errAlertsClient
	}
	r.Alerts = alertsClient

	// examine DeletionTimestamp to determine if object is under deletion
	if condition.DeletionTimestamp.IsZero() {
		if !containsString(condition.Finalizers, alertsNrqlConditionDeleteFinalizer) {
			condition.Finalizers = append(condition.Finalizers, alertsNrqlConditionDeleteFinalizer)
		}
	} else {
		// the object is being deleted
		if containsString(condition.Finalizers, alertsNrqlConditionDeleteFinalizer) {
			// catch invalid state
			if condition.Status.ConditionID == "" {
				r.Log.Info("No Condition ID set, just removing finalizer")
				condition.Finalizers = removeString(condition.Finalizers, alertsNrqlConditionDeleteFinalizer)
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
						"apiKey", interfaces.PartialAPIKey(r.apiKey),
					)
					if err.Error() == "resource not found" {
						r.Log.Info("New Relic API returned resource not found, deleting condition resource")
					} else {
						r.Log.Error(err, "Failed to delete API Condition",
							"conditionId", condition.Status.ConditionID,
							"region", condition.Spec.Region,
							"apiKey", interfaces.PartialAPIKey(r.apiKey),
						)
						return ctrl.Result{}, err
					}
				}
				// remove our finalizer from the list and update it.
				r.Log.Info("New Relic Alert condition deleted, Removing finalizer")
				condition.Finalizers = removeString(condition.Finalizers, alertsNrqlConditionDeleteFinalizer)
				if err := r.Client.Update(ctx, &condition); err != nil {
					r.Log.Error(err, "Failed to update condition after deleting New Relic Alert condition")
					return ctrl.Result{}, err
				}
			}
		}

		// stop reconciliation as the item is being deleted
		r.Log.Info("All done with condition deletion", "conditionName", condition.Spec.Name)

		return ctrl.Result{}, nil
	}

	if reflect.DeepEqual(&condition.Spec, condition.Status.AppliedSpec) {
		return ctrl.Result{}, nil
	}

	r.Log.Info("Reconciling", "condition", condition.Name)

	//check if condition has condition id
	r.checkForExistingCondition(&condition)

	r.writeNewRelicAlertCondition(ctx, req, alertsClient, condition)

	return ctrl.Result{}, nil
}

func (r *AlertsNrqlConditionReconciler) checkForExistingCondition(condition *nrv1.AlertsNrqlCondition) {
	defer r.txn.StartSegment("checkForExistingCondition").End()
	if condition.Status.ConditionID == "" {
		r.Log.Info("Checking for existing condition", "conditionName", condition.Name)
		//if no conditionId, get list of conditions and compare name
		searchParams := alerts.NrqlConditionsSearchCriteria{
			PolicyID: condition.Spec.ExistingPolicyID,
		}
		existingConditions, err := r.Alerts.SearchNrqlConditionsQuery(condition.Spec.AccountID, searchParams)
		if err != nil {
			r.Log.Error(err, "failed to get list of NRQL conditions from New Relic API",
				"conditionId", condition.Status.ConditionID,
				"region", condition.Spec.Region,
				"apiKey", interfaces.PartialAPIKey(r.apiKey),
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

func (r *AlertsNrqlConditionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.AlertClientFunc = interfaces.InitializeAlertsClient

	return ctrl.NewControllerManagedBy(mgr).
		For(&nrv1.AlertsNrqlCondition{}).
		Complete(r)
}

func (r *AlertsNrqlConditionReconciler) writeNewRelicAlertCondition(ctx context.Context, req ctrl.Request, alertsClient interfaces.NewRelicAlertsClient, condition nrv1.AlertsNrqlCondition) {
	updateInput := condition.Spec.ToNrqlConditionInput()

	if condition.Status.ConditionID != "" && !reflect.DeepEqual(&condition.Spec, condition.Status.AppliedSpec) {
		r.Log.Info("updating condition", "ConditionName", condition.Name, "API fields", updateInput)
		updatedCondition, err := alertsClient.UpdateNrqlConditionStaticMutation(condition.Spec.AccountID, condition.Status.ConditionID, updateInput)
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
		r.Log.Info("Creating condition", "ConditionName", condition.Name, "API fields", updateInput)
		createdCondition, err := alertsClient.CreateNrqlConditionStaticMutation(condition.Spec.AccountID, condition.Spec.ExistingPolicyID, updateInput)

		if err != nil {
			r.Log.Error(err, "failed to create condition",
				"conditionId", condition.Status.ConditionID,
				"region", condition.Spec.Region,
				"apiKey", interfaces.PartialAPIKey(r.apiKey),
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
}

func (r *AlertsNrqlConditionReconciler) deleteNewRelicAlertCondition(condition nrv1.AlertsNrqlCondition) error {
	defer r.txn.StartSegment("deleteNewRelicAlertCondition").End()
	r.Log.Info("Deleting condition", "conditionName", condition.Spec.Name)
	_, err := r.Alerts.DeleteConditionMutation(condition.Spec.AccountID, condition.Status.ConditionID)
	if err != nil {
		r.Log.Error(err, "Error deleting condition",
			"conditionId", condition.Status.ConditionID,
			"region", condition.Spec.Region,
			"apiKey", interfaces.PartialAPIKey(r.apiKey),
		)
		return err
	}

	return nil
}

func (r *AlertsNrqlConditionReconciler) getAPIKeyOrSecret(condition nrv1.AlertsNrqlCondition) (string, error) {
	defer r.txn.StartSegment("getAPIKeyOrSecret").End()
	if condition.Spec.APIKey != "" {
		return condition.Spec.APIKey, nil
	}

	if condition.Spec.APIKeySecret != (nrv1.NewRelicAPIKeySecret{}) {
		key := types.NamespacedName{Namespace: condition.Spec.APIKeySecret.Namespace, Name: condition.Spec.APIKeySecret.Name}
		var apiKeySecret v1.Secret
		if getErr := r.Client.Get(context.Background(), key, &apiKeySecret); getErr != nil {
			r.Log.Error(getErr, "Error retrieving secret", "secret", apiKeySecret)
			return "", getErr
		}
		return string(apiKeySecret.Data[condition.Spec.APIKeySecret.KeyName]), nil
	}

	return "", nil
}
