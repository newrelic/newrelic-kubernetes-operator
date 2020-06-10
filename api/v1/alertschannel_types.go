package v1

import (
	"encoding/json"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AlertsChannelSpec defines the desired state of AlertsChannel
type AlertsChannelSpec struct {
	ID           int                  `json:"id,omitempty"`
	Name         string               `json:"name,omitempty"`
	APIKey       string               `json:"api_key,omitempty"`
	APIKeySecret NewRelicAPIKeySecret `json:"api_key_secret,omitempty"`
	Region       string               `json:"region,omitempty"`
	Type         alerts.ChannelType   `json:"type,omitempty"`
	Links        ChannelLinks         `json:"links,omitempty"`
	// Configuration alerts.ChannelConfiguration `json:"configuration,omitempty"`  //TODO: this breaks things

}

// ChannelLinks - copy of alerts.ChannelLinks
type ChannelLinks struct {
	PolicyIDs []int `json:"policy_ids,omitempty"`
}

// AlertsChannelStatus defines the observed state of AlertsChannel
type AlertsChannelStatus struct {
	AppliedSpec *AlertsChannelSpec `json:"applied_spec"`
	ChannelID   int                `json:"Channel_id"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Created",type="boolean",JSONPath=".status.created"

// AlertsChannel is the Schema for the AlertsChannel API
type AlertsChannel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlertsChannelSpec   `json:"spec,omitempty"`
	Status AlertsChannelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AlertsChannelList contains a list of AlertsChannel
type AlertsChannelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AlertsChannel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AlertsChannel{}, &AlertsChannelList{})
}

func (in AlertsChannelSpec) APICondition() alerts.Channel {
	jsonString, _ := json.Marshal(in)
	var APICondition alerts.Channel
	json.Unmarshal(jsonString, &APICondition) //nolint
	return APICondition
}
