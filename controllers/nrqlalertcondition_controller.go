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
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nralertsv1beta1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1beta1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
)

// NrqlAlertConditionReconciler reconciles a NrqlAlertCondition object
type NrqlAlertConditionReconciler struct {
	client.Client
	Alerts interfaces.NewRelicAlertsClient
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=nr-alerts.k8s.newrelic.com,resources=nrqlalertconditions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nr-alerts.k8s.newrelic.com,resources=nrqlalertconditions/status,verbs=get;update;patch

func (r *NrqlAlertConditionReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("nrqlalertcondition", req.NamespacedName)

	r.Log.Info("Starting reconcile action")
	var condition nralertsv1beta1.NrqlAlertCondition
	err := r.Client.Get(ctx, req.NamespacedName, &condition)
	if err != nil {
		if strings.Contains(err.Error(), " not found") {
			r.Log.Info("Expected error 'not found' after condition deleted", "error", err)
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "Tried getting condition", "name", req.NamespacedName.String())
		return ctrl.Result{}, nil
	}

	deleteFinalizer := "storage.finalizers.tutorial.kubebuilder.io"

	// proto code to manually delete
	//condition.Finalizers = removeString(condition.Finalizers, deleteFinalizer)

	//examine DeletionTimestamp to determine if object is under deletion
	if condition.DeletionTimestamp.IsZero() {
		if !containsString(condition.Finalizers, deleteFinalizer) {
			condition.Finalizers = append(condition.Finalizers, deleteFinalizer)
		}
	} else {
		// The object is being deleted
		if containsString(condition.Finalizers, deleteFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteNewRelicAlertCondition(condition); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}
			// remove our finalizer from the list and update it.
			r.Log.Info("New Relic Alert condition deleted, Removing finalizer")
			condition.Finalizers = removeString(condition.Finalizers, deleteFinalizer)
			if err := r.Client.Update(ctx, &condition); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		r.Log.Info("All done with condition deletion", "conditionName", condition.Spec.Name)

		return ctrl.Result{}, nil
	}

	if reflect.DeepEqual(&condition.Spec, condition.Status.AppliedSpec) {
		return ctrl.Result{}, nil
	}

	r.Log.Info("Reconciling", "condition", condition)

	//check if condition has condition id
	r.checkForExistingCondition(&condition)

	APICondition := condition.Spec.APICondition()
	r.Log.Info("Trying to create or update condition", "API fields", APICondition)

	if condition.Status.ConditionID != 0 && !reflect.DeepEqual(&condition.Spec, condition.Status.AppliedSpec) {
		r.Log.Info("updating condition")
		APICondition.ID = condition.Status.ConditionID
		updatedCondition, err := r.Alerts.UpdateNrqlCondition(APICondition)
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
		createdCondition, err := r.Alerts.CreateNrqlCondition(APICondition)
		if err != nil {
			r.Log.Error(err, "failed to create condition")
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

func (r *NrqlAlertConditionReconciler) checkForExistingCondition(condition *nralertsv1beta1.NrqlAlertCondition) {
	if condition.Status.ConditionID == 0 {
		r.Log.Info("Checking for existing condition", "conditionName", condition.Name)
		//if no conditionId, get list of conditions and compare name
		existingConditions, err := r.Alerts.ListNrqlConditions(condition.Spec.ExistingPolicyId)
		r.Log.Info("existingConditions", "existingConditions", existingConditions)
		if err != nil {
			r.Log.Error(err, "failed to get list of NRQL conditions from New Relic API")
		} else {
			for _, existingCondition := range existingConditions {
				if existingCondition.Name == condition.Spec.Name {
					r.Log.Info("Matched on existing condition, updating ConditionId", "conditionId", existingCondition.ID)
					condition.Status.ConditionID = existingCondition.ID
				}
				break
			}
		}

	}
}

func (r *NrqlAlertConditionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nralertsv1beta1.NrqlAlertCondition{}).
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

func (r *NrqlAlertConditionReconciler) deleteNewRelicAlertCondition(condition nralertsv1beta1.NrqlAlertCondition) error {
	r.Log.Info("Deleting condition", "conditionName", condition.Spec.Name)
	_, err := r.Alerts.DeleteNrqlCondition(condition.Status.ConditionID)
	if err != nil {
		r.Log.Info("Error thrown", "error", err)
		return err
	}
	return nil
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
