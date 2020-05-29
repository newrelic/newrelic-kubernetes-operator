package v1

import (
	"hash"

	"github.com/davecgh/go-spew/spew"
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
