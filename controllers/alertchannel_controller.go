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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
)

// AlertChannelReconciler reconciles a AlertChannel object
type AlertChannelReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	AlertClientFunc func(string, string) (interfaces.NewRelicAlertsClient, error)
	apiKey          string
	Alerts          interfaces.NewRelicAlertsClient
	ctx             context.Context
}

// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=alertchannel,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=alertchannel/status,verbs=get;update;patch

func (r *AlertChannelReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	r.ctx = context.Background()
	_ = r.Log.WithValues("alertChannel", req.NamespacedName)

	var alertChannel nrv1.AlertChannel
	err := r.Client.Get(r.ctx, req.NamespacedName, &alertChannel)
	if err != nil {
		if strings.Contains(err.Error(), " not found") {
			r.Log.Info("AlertChannel 'not found' after being deleted. This is expected and no cause for alarm", "error", err)
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "Failed to GET alertChannel", "name", req.NamespacedName.String())
		return ctrl.Result{}, nil
	}
	r.Log.Info("Starting reconcile action")
	r.Log.Info("alertChannel", "alertChannel.Spec", alertChannel.Spec, "alertChannel.status.applied", alertChannel.Status.AppliedSpec)

	r.apiKey = r.getAPIKeyOrSecret(alertChannel)

	if r.apiKey == "" {
		return ctrl.Result{}, errors.New("api key is blank")
	}
	//initial alertsClient
	alertsClient, errAlertsClient := r.AlertClientFunc(r.apiKey, alertChannel.Spec.Region)
	if errAlertsClient != nil {
		r.Log.Error(errAlertsClient, "Failed to create AlertsClient")
		return ctrl.Result{}, errAlertsClient
	}
	r.Alerts = alertsClient

	deleteFinalizer := "alertchannels.finalizers.nr.k8s.newrelic.com"

	//examine DeletionTimestamp to determine if object is under deletion
	if alertChannel.DeletionTimestamp.IsZero() {
		if !containsString(alertChannel.Finalizers, deleteFinalizer) {
			alertChannel.Finalizers = append(alertChannel.Finalizers, deleteFinalizer)
		}
	} else {
		return r.deleteAlertChannel(r.ctx, &alertChannel, deleteFinalizer)
	}

	if reflect.DeepEqual(&alertChannel.Spec, alertChannel.Status.AppliedSpec) {
		return ctrl.Result{}, nil
	}

	r.Log.Info("Reconciling", "alertChannel", alertChannel.Name)

	r.checkForExistingAlertChannel(&alertChannel)

	if alertChannel.Status.ChannelID != 0 {
		err := r.updateAlertChannel(&alertChannel)
		if err != nil {
			r.Log.Error(err, "error updating alertChannel")
			return ctrl.Result{}, err
		}
	} else {
		err := r.createAlertChannel(&alertChannel)
		if err != nil {
			r.Log.Error(err, "Error creating alertChannel")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *AlertChannelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nrv1.AlertChannel{}).
		Complete(r)
}

func (r *AlertChannelReconciler) getAPIKeyOrSecret(alertchannel nrv1.AlertChannel) string {

	if alertchannel.Spec.APIKey != "" {
		return alertchannel.Spec.APIKey
	}
	if alertchannel.Spec.APIKeySecret != (nrv1.NewRelicAPIKeySecret{}) {
		key := types.NamespacedName{Namespace: alertchannel.Spec.APIKeySecret.Namespace, Name: alertchannel.Spec.APIKeySecret.Name}
		var apiKeySecret v1.Secret
		getErr := r.Client.Get(context.Background(), key, &apiKeySecret)
		if getErr != nil {
			r.Log.Error(getErr, "Failed to retrieve secret", "secret", apiKeySecret)
			return ""
		}
		return string(apiKeySecret.Data[alertchannel.Spec.APIKeySecret.KeyName])
	}
	return ""
}

func (r *AlertChannelReconciler) deleteAlertChannel(ctx context.Context, alertChannel *nrv1.AlertChannel, deleteFinalizer string) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *AlertChannelReconciler) createAlertChannel(alertchannel *nrv1.AlertChannel) error {
	return nil
}

func (r *AlertChannelReconciler) updateAlertChannel(alertchannel *nrv1.AlertChannel) error {
	return nil
}

func (r *AlertChannelReconciler) checkForExistingAlertChannel(alertChannel *nrv1.AlertChannel) {}
