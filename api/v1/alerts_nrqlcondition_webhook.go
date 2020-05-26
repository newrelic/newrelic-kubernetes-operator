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
	"context"
	"errors"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
)

func (r *AlertsNrqlCondition) SetupWebhookWithManager(mgr ctrl.Manager) error {
	alertClientFunc = interfaces.InitializeAlertsClient
	k8Client = mgr.GetClient()
	ctx = context.Background()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-nr-k8s-newrelic-com-v1-alertsnrqlcondition,mutating=true,failurePolicy=fail,groups=nr.k8s.newrelic.com,resources=alertsnrqlconditions,verbs=create;update,versions=v1,name=malertsnrqlcondition.kb.io

var _ webhook.Defaulter = &AlertsNrqlCondition{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *AlertsNrqlCondition) Default() {
	log.Info("default", "name", r.Name)

	if r.Status.AppliedSpec == nil {
		log.Info("Setting null Applied Spec to empty interface")
		r.Status.AppliedSpec = &AlertsNrqlConditionSpec{}
	}
	log.Info("r.Status.AppliedSpec after", "r.Status.AppliedSpec", r.Status.AppliedSpec)
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-nr-k8s-newrelic-com-v1-alertsnrqlcondition,mutating=false,failurePolicy=fail,groups=nr.k8s.newrelic.com,resources=alertsnrqlconditions,versions=v1,name=valertsnrqlcondition.kb.io

var _ webhook.Validator = &AlertsNrqlCondition{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *AlertsNrqlCondition) ValidateCreate() error {
	log.Info("validate create", "name", r.Name)
	//TODO this should write this value TO a new secret so code path always reads from a secret
	err := r.CheckForAPIKeyOrSecret()
	if err != nil {
		return err
	}

	err = r.CheckRequiredFields()
	if err != nil {
		return err
	}
	return r.CheckExistingPolicyID()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AlertsNrqlCondition) ValidateUpdate(old runtime.Object) error {
	log.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AlertsNrqlCondition) ValidateDelete() error {
	log.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *AlertsNrqlCondition) CheckExistingPolicyID() error {
	log.Info("Checking existing", "policyId", r.Spec.ExistingPolicyID)
	var apiKey string
	if r.Spec.APIKey == "" {
		key := types.NamespacedName{Namespace: r.Spec.APIKeySecret.Namespace, Name: r.Spec.APIKeySecret.Name}
		var apiKeySecret v1.Secret
		getErr := k8Client.Get(ctx, key, &apiKeySecret)
		if getErr != nil {
			log.Error(getErr, "Error getting secret")
			return getErr
		}
		apiKey = string(apiKeySecret.Data[r.Spec.APIKeySecret.KeyName])

	} else {
		apiKey = r.Spec.APIKey
	}

	alertsClient, errAlertClient := alertClientFunc(apiKey, r.Spec.Region)
	if errAlertClient != nil {
		log.Error(errAlertClient, "failed to get policy",
			"policyId", r.Spec.ExistingPolicyID,
			"API Key", interfaces.PartialAPIKey(apiKey),
			"region", r.Spec.Region,
		)
		return errAlertClient
	}
	alertPolicy, errAlertPolicy := alertsClient.GetPolicy(r.Spec.ExistingPolicyID)
	if errAlertPolicy != nil {
		log.Error(errAlertPolicy, "failed to get policy",
			"policyId", r.Spec.ExistingPolicyID,
			"API Key", interfaces.PartialAPIKey(apiKey),
			"region", r.Spec.Region,
		)
		return errAlertPolicy
	}
	if alertPolicy.ID != r.Spec.ExistingPolicyID {
		log.Info("Alert policy returned by the API failed to match provided policy ID")
		return errors.New("alert policy returned by API did not match")
	}
	return nil
}

func (r *AlertsNrqlCondition) CheckForAPIKeyOrSecret() error {
	if r.Spec.APIKey != "" {
		return nil
	}
	if r.Spec.APIKeySecret != (NewRelicAPIKeySecret{}) {
		if r.Spec.APIKeySecret.Name != "" && r.Spec.APIKeySecret.Namespace != "" && r.Spec.APIKeySecret.KeyName != "" {
			return nil
		}
	}
	return errors.New("either api_key or api_key_secret must be set")
}

func (r *AlertsNrqlCondition) CheckRequiredFields() error {

	missingFields := []string{}
	if r.Spec.Region == "" {
		missingFields = append(missingFields, "region")
	}
	if r.Spec.ExistingPolicyID == 0 {
		missingFields = append(missingFields, "existing_policy_id")
	}
	if len(missingFields) > 0 {
		return errors.New(strings.Join(missingFields, " and ") + " must be set")
	}
	return nil
}
