package v1

import (
	"encoding/json"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AlertChannelSpec defines the desired state of AlertChannel
type AlertChannelSpec struct {
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

// AlertChannelStatus defines the observed state of AlertChannel
type AlertChannelStatus struct {
	AppliedSpec *AlertChannelSpec `json:"applied_spec"`
	ChannelID   int               `json:"Channel_id"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Created",type="boolean",JSONPath=".status.created"

// AlertChannel is the Schema for the AlertChannel API
type AlertChannel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlertChannelSpec   `json:"spec,omitempty"`
	Status AlertChannelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AlertChannelList contains a list of AlertChannel
type AlertChannelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AlertChannel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AlertChannel{}, &AlertChannelList{})
}

func (in AlertChannelSpec) APICondition() alerts.Channel {
	jsonString, _ := json.Marshal(in)
	var APICondition alerts.Channel
	json.Unmarshal(jsonString, &APICondition) //nolint
	return APICondition
}
