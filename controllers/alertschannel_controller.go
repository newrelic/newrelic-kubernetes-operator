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
	"sort"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	newrelic "github.com/newrelic/go-agent/v3/newrelic"
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
	NewRelicAgent   newrelic.Application
	txn             *newrelic.Transaction
}

// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=alertschannel,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nr.k8s.newrelic.com,resources=alertschannel/status,verbs=get;update;patch

//Reconcile - Main processing loop for AlertsChannel reconciliation
func (r *AlertsChannelReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var alertsChannel nrv1.AlertsChannel

	r.ctx = context.Background()
	r.Log.WithValues("alertsChannel", req.NamespacedName)

	r.txn = r.NewRelicAgent.StartTransaction("Reconcile/Alerts/AlertsPolicy")
	defer r.txn.End()

	err := r.Client.Get(r.ctx, req.NamespacedName, &alertsChannel)
	if err != nil {
		if strings.Contains(err.Error(), " not found") {
			r.Log.Info("AlertsChannel 'not found' after being deleted. This is expected and no cause for alarm", "error", err)
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "Failed to GET alertsChannel", "name", req.NamespacedName.String())
		return ctrl.Result{}, nil
	}

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
	defer r.txn.StartSegment("getAPIKeyOrSecret").End()
	if alertschannel.Spec.APIKey != "" {
		return alertschannel.Spec.APIKey
	}

	if alertschannel.Spec.APIKeySecret != (nrv1.NewRelicAPIKeySecret{}) {
		var apiKeySecret v1.Secret

		key := types.NamespacedName{Namespace: alertschannel.Spec.APIKeySecret.Namespace, Name: alertschannel.Spec.APIKeySecret.Name}

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
	defer r.txn.StartSegment("deleteAlertsChannel").End()
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
	defer r.txn.StartSegment("createAlertsChannel").End()
	r.Log.Info("Creating AlertsChannel", "name", alertsChannel.Name, "ChannelName", alertsChannel.Spec.Name)
	APIChannel := alertsChannel.Spec.APIChannel()

	r.Log.Info("API Payload before calling NR API", "APIChannel", APIChannel)

	createdChannel, err := r.Alerts.CreateChannel(APIChannel)
	if err != nil {
		r.Log.Error(err, "Error creating AlertsChannel"+alertsChannel.Name)
		return err
	}

	alertsChannel.Status.ChannelID = createdChannel.ID

	// Now create the links to policies
	allPolicyIDs, err := r.getAllPolicyIDs(&alertsChannel.Spec)

	if err != nil {
		r.Log.Error(err, "Error getting list of policyIds")
		return err
	}

	for _, policyID := range allPolicyIDs {
		policyChannels, errUpdatePolicies := r.Alerts.UpdatePolicyChannels(policyID, []int{createdChannel.ID})
		if errUpdatePolicies != nil {
			r.Log.Error(errUpdatePolicies, "error updating policyAlertsChannels", "policyID", policyID, "conditionID", createdChannel.ID, "policyChannels", policyChannels)
		} else {
			alertsChannel.Status.AppliedPolicyIDs = append(alertsChannel.Status.AppliedPolicyIDs, policyID)
		}
	}

	alertsChannel.Status.AppliedSpec = &alertsChannel.Spec
	errClientUpdate := r.Client.Update(r.ctx, alertsChannel)

	if errClientUpdate != nil {
		r.Log.Error(errClientUpdate, "Error updating channel status", "name", alertsChannel.Name, "Namespace", alertsChannel.Namespace)
		return errClientUpdate
	}

	return nil
}

func (r *AlertsChannelReconciler) updateAlertsChannel(alertsChannel *nrv1.AlertsChannel) error {
	defer r.txn.StartSegment("updateAlertsChannel").End()
	r.Log.Info("Updating AlertsChannel", "name", alertsChannel.Name, "ChannelName", alertsChannel.Spec.Name)

	//Check to see if update is needed
	AppliedPolicyIDs, AppliedErr := r.getAllPolicyIDs(alertsChannel.Status.AppliedSpec)

	if AppliedErr != nil {
		r.Log.Error(AppliedErr, "Error getting list of AppliedPolicyIds")
		return AppliedErr
	}

	IncomingPolicyIDs, incomingErr := r.getAllPolicyIDs(&alertsChannel.Spec)

	if incomingErr != nil {
		r.Log.Error(incomingErr, "Error getting list of AppliedPolicyIds")
		return incomingErr
	}

	r.Log.Info("Updating list of policies attached to AlertsChannel",
		"policyIDs", IncomingPolicyIDs,
		"AppliedPolicyIDs", AppliedPolicyIDs,
	)

	processedPolicyIDs := make(map[int]bool)

	for _, incomingPolicyID := range IncomingPolicyIDs {
		processedPolicyIDs[incomingPolicyID] = false
	}

	for _, appliedPolicyID := range AppliedPolicyIDs {
		if _, ok := processedPolicyIDs[appliedPolicyID]; ok {
			processedPolicyIDs[appliedPolicyID] = true
		} else {
			r.Log.Info("Need to delete link to", "policyId", appliedPolicyID)
			PolicyChannels, err := r.Alerts.DeletePolicyChannel(appliedPolicyID, alertsChannel.Status.ChannelID)
			if err != nil {
				r.Log.Error(err, "error updating policyAlertsChannels",
					"policyID", appliedPolicyID,
					"conditionID", alertsChannel.Status.ChannelID,
					"PolicyChannels", PolicyChannels,
				)
			}
		}
	}

	//Clear out AppliedPolicyIds
	alertsChannel.Status.AppliedPolicyIDs = []int{}

	for policyID, processed := range processedPolicyIDs {
		r.Log.Info("processing ", "policyID", policyID, ":processed", processed)
		alertsChannel.Status.AppliedPolicyIDs = append(alertsChannel.Status.AppliedPolicyIDs, policyID)

		if !processed {
			r.Log.Info("need to add ", "policyID", policyID)

			policyChannels, err := r.Alerts.UpdatePolicyChannels(policyID, []int{alertsChannel.Status.ChannelID})
			if err != nil {
				r.Log.Error(err, "error updating policyAlertsChannels",
					"policyID", policyID,
					"conditionID", alertsChannel.Status.ChannelID,
					"policyChannels", policyChannels,
				)
				r.Log.Info("policyChannels", "", policyChannels)
			}
		}
	}

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
	defer r.txn.StartSegment("checkForExistingAlertsChannel").End()
	r.Log.Info("Checking for existing Channels matching name: " + alertsChannel.Spec.Name)
	retrievedChannels, err := r.Alerts.ListChannels()

	if err != nil {
		r.Log.Error(err, "error retrieving list of Channels")
		return
	}

	// need to delete all non-matching spec channels
	for _, channel := range retrievedChannels {
		if channel.Name == alertsChannel.Spec.Name {
			channelID := channel.ID
			channel.ID = 0
			APIChannel := alertsChannel.Spec.APIChannel()

			if reflect.DeepEqual(&APIChannel, channel) {
				r.Log.Info("Found matching Alerts Channel name from the New Relic API", "ID", channel.ID)
				alertsChannel.Status.ChannelID = channelID

				alertsChannel.Status.AppliedSpec = &alertsChannel.Spec
			}

			r.Log.Info("Found non matching channel so need to delete and create channel")

			_, err = r.Alerts.DeleteChannel(channelID)
			if err != nil {
				r.Log.Error(err, "Error deleting non-matching AlertsChannel via New Relic API")
				continue
			}
		}
	}
}

func (r *AlertsChannelReconciler) getAllPolicyIDs(alertsChannelSpec *nrv1.AlertsChannelSpec) (policyIDs []int, err error) {
	defer r.txn.StartSegment("getAllPolicyIDs").End()
	var retrievedPolicies []alerts.Policy
	policyIDMap := make(map[int]bool)

	for _, policyID := range alertsChannelSpec.Links.PolicyIDs {
		policyIDMap[policyID] = true
	}

	if len(alertsChannelSpec.Links.PolicyNames) > 0 {
		for _, policyName := range alertsChannelSpec.Links.PolicyNames {
			alertParams := &alerts.ListPoliciesParams{
				Name: policyName,
			}

			retrievedPolicies, err = r.Alerts.ListPolicies(alertParams)
			if err != nil {
				r.Log.Error(err, "Error getting list of policies")
				return
			}

			for _, APIPolicy := range retrievedPolicies {
				if policyName == APIPolicy.Name {
					r.Log.Info("Found match of "+policyName, "policyId", APIPolicy.ID)
					policyIDMap[APIPolicy.ID] = true
				}
			}
		}
	}

	if len(alertsChannelSpec.Links.PolicyKubernetesObjects) > 0 {
		r.Log.Info("Getting PolicyIds from PolicyKubernetesObjects",
			"PolicyKubernetesObjects", alertsChannelSpec.Links.PolicyKubernetesObjects,
		)

		for _, policyK8s := range alertsChannelSpec.Links.PolicyKubernetesObjects {
			key := types.NamespacedName{
				Namespace: policyK8s.Namespace,
				Name:      policyK8s.Name,
			}

			var k8sPolicy nrv1.AlertsPolicy

			err = r.Client.Get(context.Background(), key, &k8sPolicy)
			if err != nil {
				r.Log.Error(err, "Failed to retrieve policy", "k8sPolicy", key)
				return
			}

			policyID, errInt := strconv.Atoi(k8sPolicy.Status.PolicyID)
			if errInt != nil {
				r.Log.Error(errInt, "Failed to parse policyID as an int")
				err = errInt
			}

			if policyID != 0 {
				policyIDMap[policyID] = true
			} else {
				r.Log.Info("Retrieved policy " + policyK8s.Name + " but ID was blank")
				err = errors.New("Retrieved policy " + policyK8s.Name + " but ID was blank")
				return
			}
		}
	}

	for policyID := range policyIDMap {
		policyIDs = append(policyIDs, policyID)
	}
	sort.Ints(policyIDs)

	return
}
