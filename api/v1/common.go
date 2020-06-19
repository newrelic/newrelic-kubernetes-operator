package v1

import (
	"errors"
	"hash"

	"github.com/davecgh/go-spew/spew"
	"github.com/newrelic/newrelic-client-go/pkg/region"
)

// DeepHashObject writes specified object to hash using the spew library
// which follows pointers and prints actual values of the nested objects
// ensuring the hash does not change when a pointer changes.
func DeepHashObject(hasher hash.Hash, objectToWrite interface{}) {
	hasher.Reset()
	printer := spew.ConfigState{
		Indent:         " ",
		SortKeys:       true,
		DisableMethods: true,
		SpewKeys:       true,
	}
	printer.Fprintf(hasher, "%#v", objectToWrite)
}

// AlertConditionTerm represents the terms of a New Relic alert condition.
type AlertConditionTerm struct {
	Duration            string `json:"duration,omitempty"`
	Operator            string `json:"operator,omitempty"`
	Priority            string `json:"priority,omitempty"`
	Threshold           string `json:"threshold"`
	TimeFunction        string `json:"time_function,omitempty"`
	ViolationCloseTimer int    `json:"violation_close_timer,omitempty"`
}

type NewRelicAPIKeySecret struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	KeyName   string `json:"key_name,omitempty"`
}

//ValidRegion - returns true if a valid region is passed
func ValidRegion(input string) bool {
	_, err := region.Parse(input)
	if err != nil {
		return false
	} else if input == "" {
		return false
	}

	return true
}

//CheckForAPIKeyOrSecret - returns error if a API KEY or k8 secret is not passed in
func CheckForAPIKeyOrSecret(apiKey string, secret NewRelicAPIKeySecret) error {
	if apiKey != "" {
		return nil
	}

	if secret != (NewRelicAPIKeySecret{}) {
		if secret.Name != "" && secret.Namespace != "" && secret.KeyName != "" {
			return nil
		}
	}

	return errors.New("either api_key or api_key_secret must be set")
}
