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

	"github.com/newrelic/newrelic-client-go/pkg/alerts"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
)

// AlertsChannelReconciler reconciles a AlertsChannel object
type AlertsChannelReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	AlertClientFunc func(string, string) (interfaces.NewRelicAlertsClient, error)
	apiKey          string
	Alerts          interfaces.NewRelicAlertsClient
	ctx             context.Context
}

// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=alertschannel,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=alertschannel/status,verbs=get;update;patch

//Reconcile - Main processing loop for AlertsChannel reconciliation
func (r *AlertsChannelReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	r.ctx = context.Background()
	_ = r.Log.WithValues("alertsChannel", req.NamespacedName)

	var alertsChannel nrv1.AlertsChannel
	err := r.Client.Get(r.ctx, req.NamespacedName, &alertsChannel)
	if err != nil {
		if strings.Contains(err.Error(), " not found") {
			r.Log.Info("AlertsChannel 'not found' after being deleted. This is expected and no cause for alarm", "error", err)
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "Failed to GET alertsChannel", "name", req.NamespacedName.String())
		return ctrl.Result{}, nil
	}
	r.Log.Info("Starting reconcile action")
	r.Log.Info("alertsChannel", "alertsChannel.Spec", alertsChannel.Spec, "alertsChannel.status.applied", alertsChannel.Status.AppliedSpec)

	r.apiKey = r.getAPIKeyOrSecret(alertsChannel)

	if r.apiKey == "" {
		return ctrl.Result{}, errors.New("api key is blank")
	}
	//initial alertsClient
	alertsClient, errAlertsClient := r.AlertClientFunc(r.apiKey, alertsChannel.Spec.Region)
	if errAlertsClient != nil {
		r.Log.Error(errAlertsClient, "Failed to create AlertsClient")
		return ctrl.Result{}, errAlertsClient
	}
	r.Alerts = alertsClient

	deleteFinalizer := "alertschannels.finalizers.nr.k8s.newrelic.com"

	//examine DeletionTimestamp to determine if object is under deletion
	if alertsChannel.DeletionTimestamp.IsZero() {

		if !containsString(alertsChannel.Finalizers, deleteFinalizer) {
			alertsChannel.Finalizers = append(alertsChannel.Finalizers, deleteFinalizer)
		}
	} else {
		r.Log.Info("K8s resource deletion")
		err := r.deleteAlertsChannel(&alertsChannel, deleteFinalizer)
		if err != nil {
			r.Log.Error(err, "error deleting channel", "name", alertsChannel.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if reflect.DeepEqual(&alertsChannel.Spec, alertsChannel.Status.AppliedSpec) {
		return ctrl.Result{}, nil
	}

	r.Log.Info("Reconciling", "alertsChannel", alertsChannel.Name)

	r.checkForExistingAlertsChannel(&alertsChannel)

	if alertsChannel.Status.ChannelID != 0 {
		err := r.updateAlertsChannel(&alertsChannel)
		if err != nil {
			r.Log.Error(err, "error updating alertsChannel")
			return ctrl.Result{}, err
		}
	} else {
		err := r.createAlertsChannel(&alertsChannel)
		if err != nil {
			r.Log.Error(err, "Error creating alertsChannel")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

//SetupWithManager - Sets up Controller for AlertsChannel
func (r *AlertsChannelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nrv1.AlertsChannel{}).
		Complete(r)
}

func (r *AlertsChannelReconciler) getAPIKeyOrSecret(alertschannel nrv1.AlertsChannel) string {

	if alertschannel.Spec.APIKey != "" {
		return alertschannel.Spec.APIKey
	}
	if alertschannel.Spec.APIKeySecret != (nrv1.NewRelicAPIKeySecret{}) {
		key := types.NamespacedName{Namespace: alertschannel.Spec.APIKeySecret.Namespace, Name: alertschannel.Spec.APIKeySecret.Name}
		var apiKeySecret v1.Secret
		getErr := r.Client.Get(context.Background(), key, &apiKeySecret)
		if getErr != nil {
			r.Log.Error(getErr, "Failed to retrieve secret", "secret", apiKeySecret)
			return ""
		}
		return string(apiKeySecret.Data[alertschannel.Spec.APIKeySecret.KeyName])
	}
	return ""
}

func (r *AlertsChannelReconciler) deleteAlertsChannel(alertsChannel *nrv1.AlertsChannel, deleteFinalizer string) (err error) {

	r.Log.Info("Deleting AlertsChannel", "name", alertsChannel.Name, "ChannelName", alertsChannel.Spec.Name)

	if alertsChannel.Status.ChannelID != 0 {
		_, err = r.Alerts.DeleteChannel(alertsChannel.Status.ChannelID)
		if err != nil {
			r.Log.Error(err, "error deleting AlertsChannel", "name", alertsChannel.Name, "ChannelName", alertsChannel.Spec.Name)
		}

	}
	// Now remove finalizer
	alertsChannel.Finalizers = removeString(alertsChannel.Finalizers, deleteFinalizer)

	err = r.Client.Update(r.ctx, alertsChannel)
	if err != nil {
		r.Log.Error(err, "tried updating condition status", "name", alertsChannel.Name, "Namespace", alertsChannel.Namespace)
		return err
	}
	return nil
}

func (r *AlertsChannelReconciler) createAlertsChannel(alertsChannel *nrv1.AlertsChannel) error {
	r.Log.Info("Creating AlertsChannel", "name", alertsChannel.Name, "ChannelName", alertsChannel.Spec.Name)
	APIChannel := alertsChannel.Spec.APIChannel()

	APIChannel.Links.PolicyIDs = r.getAllPolicyIDs(&alertsChannel.Spec)
	createdCondition, err := r.Alerts.CreateChannel(APIChannel)
	if err != nil {
		r.Log.Error(err, "Error creating AlertsChannel"+alertsChannel.Name)
		return err
	}
	alertsChannel.Status.ChannelID = createdCondition.ID
	alertsChannel.Status.AppliedSpec = &alertsChannel.Spec
	err = r.Client.Update(r.ctx, alertsChannel)
	if err != nil {
		r.Log.Error(err, "Tried updating channel status", "name", alertsChannel.Name, "Namespace", alertsChannel.Namespace)
		return err
	}
	return nil
}

func (r *AlertsChannelReconciler) updateAlertsChannel(alertsChannel *nrv1.AlertsChannel) error {
	r.Log.Info("Updating AlertsChannel", "name", alertsChannel.Name, "ChannelName", alertsChannel.Spec.Name)

	//Check to see if update is needed
	AppliedPolicyIDs := r.getAllPolicyIDs(alertsChannel.Status.AppliedSpec)

	IncomingPolicyIDs := r.getAllPolicyIDs(&alertsChannel.Spec)

	r.Log.Info("Updating list of policies attached to AlertsChannel", "policyIDs", IncomingPolicyIDs, "AppliedPolicyIDs", AppliedPolicyIDs)

	//TODO: Need to add this work
	// if AppliedPolicyIDs != IncomingPolicyIDs {
	// 	r.Log.Info("Updating list of policies attached to AlertsChannel", "policyIDs", IncomingPolicyIDs)

	// }

	// Now update the AppliedSpec and the k8s object
	alertsChannel.Status.AppliedSpec = &alertsChannel.Spec
	err := r.Client.Update(r.ctx, alertsChannel)
	if err != nil {
		r.Log.Error(err, "Tried updating channel status", "name", alertsChannel.Name, "Namespace", alertsChannel.Namespace)
		return err
	}

	return nil
}

func (r *AlertsChannelReconciler) checkForExistingAlertsChannel(alertsChannel *nrv1.AlertsChannel) {
	r.Log.Info("Checking for existing Channels matching name: " + alertsChannel.Spec.Name)
	retrievedChannels, err := r.Alerts.ListChannels()
	if err != nil {
		r.Log.Error(err, "error retrieving list of Channels")
		return
	}
	for _, channel := range retrievedChannels {
		if channel.Name == alertsChannel.Spec.Name {
			r.Log.Info("Found matching Alerts Channel name from the New Relic API", "ID", channel.ID)
			alertsChannel.Status.ChannelID = channel.ID
			return
		}
	}
}

func (r *AlertsChannelReconciler) getAllPolicyIDs(alertsChannelSpec *nrv1.AlertsChannelSpec) (policyIDs []int) {
	policyIDs = append(policyIDs, alertsChannelSpec.Links.PolicyIDs...)

	if len(alertsChannelSpec.Links.PolicyNames) > 0 {
		r.Log.Info("Getting PolicyIds from PolicyNames", "PolicyNames", alertsChannelSpec.Links.PolicyNames)
		alertParams := &alerts.ListPoliciesParams{
			Name: alertsChannelSpec.Links.PolicyNames[0],
		}
		retrievedPolicies, err := r.Alerts.ListPolicies(alertParams)
		if err != nil {
			r.Log.Error(err, "Error getting list of policies")
		}
		for _, policyName := range alertsChannelSpec.Links.PolicyNames {
			for _, APIPolicy := range retrievedPolicies {
				if policyName == APIPolicy.Name {
					r.Log.Info("Found match of "+policyName, "policyId", APIPolicy.ID)
					policyIDs = append(policyIDs, APIPolicy.ID)
				}
			}
		}

	}

	if len(alertsChannelSpec.Links.PolicyKubernetesObjects) > 0 {
		r.Log.Info("Getting PolicyIds from PolicyKubernetesObjects", "PolicyKubernetesObjects", alertsChannelSpec.Links.PolicyKubernetesObjects)
		for _, policyK8s := range alertsChannelSpec.Links.PolicyKubernetesObjects {

			key := types.NamespacedName{
				Namespace: policyK8s.Namespace,
				Name:      policyK8s.Name,
			}
			var k8sPolicy nrv1.AlertsPolicy
			getErr := r.Client.Get(context.Background(), key, &k8sPolicy)
			if getErr != nil {
				r.Log.Error(getErr, "Failed to retrieve policy", "k8sPolicy", key)
			}
			if k8sPolicy.Status.PolicyID != 0 {
				policyIDs = append(policyIDs, k8sPolicy.Status.PolicyID)
			} else {
				r.Log.Info("Retrieved policy " + policyK8s.Name + "but ID was 0")
			}

		}
	}

	r.Log.Info("Full list of PolicyIs to a to Channel", "policyIDs", policyIDs)

	return
}
