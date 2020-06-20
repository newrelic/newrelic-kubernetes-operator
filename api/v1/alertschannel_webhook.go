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

package v1

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"

	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
)

// log is for logging in this package.
var (
	alertschannellog = logf.Log.WithName("alertschannel-resource")
)

// SetupWebhookWithManager - instantiates the Webhook
func (r *AlertsChannel) SetupWebhookWithManager(mgr ctrl.Manager) error {
	alertClientFunc = interfaces.InitializeAlertsClient
	k8Client = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-nr-k8s-newrelic-com-v1-alertschannel,mutating=true,failurePolicy=fail,groups=nr.k8s.newrelic.com,resources=alertschannels,verbs=create;update,versions=v1,name=malertschannel.kb.io,sideEffects=None

var _ webhook.Defaulter = &AlertsChannel{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *AlertsChannel) Default() {
	alertschannellog.Info("default", "name", r.Name)

	if r.Status.AppliedSpec == nil {
		log.Info("Setting null Applied Spec to empty interface")
		r.Status.AppliedSpec = &AlertsChannelSpec{}
	}

	if r.Status.AppliedPolicyIDs == nil {
		log.Info("Setting null AppliedPolicyIDs to empty interface")
		r.Status.AppliedPolicyIDs = []int{}
	}
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-nr-k8s-newrelic-com-v1-alertschannel,mutating=false,failurePolicy=fail,groups=nr.k8s.newrelic.com,resources=alertschannels,versions=v1,name=valertschannel.kb.io,sideEffects=None

var _ webhook.Validator = &AlertsChannel{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *AlertsChannel) ValidateCreate() error {
	alertschannellog.Info("validate create", "name", r.Name)

	return r.ValidateAlertsChannel()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AlertsChannel) ValidateUpdate(old runtime.Object) error {
	alertschannellog.Info("validate update", "name", r)

	return r.ValidateAlertsChannel()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AlertsChannel) ValidateDelete() error {
	alertschannellog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

// ValidateAlertsChannel - Validates create/update of AlertsChannel
func (r *AlertsChannel) ValidateAlertsChannel() error {
	err := CheckForAPIKeyOrSecret(r.Spec.APIKey, r.Spec.APIKeySecret)
	if err != nil {
		return err
	}

	if !ValidRegion(r.Spec.Region) {
		return errors.New("Invalid region set, value was: " + r.Spec.Region)
	}

	var invalidAttributes InvalidAttributeSlice

	r.ValidateType()
	invalidAttributes = append(invalidAttributes, r.ValidateType()...)

	if len(invalidAttributes) > 0 {
		return errors.New("error with invalid attributes: \n" + invalidAttributes.errorString())
	}

	return nil
}

//ValidateType - Validates the Type attribute
func (r *AlertsChannel) ValidateType() InvalidAttributeSlice {
	switch r.Spec.Type {
	case string(alerts.ChannelTypes.Email),
		string(alerts.ChannelTypes.OpsGenie),
		string(alerts.ChannelTypes.PagerDuty),
		string(alerts.ChannelTypes.Slack),
		string(alerts.ChannelTypes.User),
		string(alerts.ChannelTypes.VictorOps),
		string(alerts.ChannelTypes.Webhook):

		return []invalidAttribute{}
	default:
		alertschannellog.Info("Invalid Type attribute", "Type", r.Spec.Type)

		return []invalidAttribute{{attribute: "Type", value: r.Spec.Type}}
	}
}
