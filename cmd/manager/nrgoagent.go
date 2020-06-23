package main

import (
	"fmt"

	"github.com/newrelic/go-agent/v3/newrelic"
)

//InitializeNRAgent - Initializes the NR Agent
func InitializeNRAgent() newrelic.Application {

	app, err := newrelic.NewApplication(
		//Set a default name which will be overridden by ENV values if provided
		newrelic.ConfigAppName("New Relic Kubernetes Operator"),
		newrelic.ConfigFromEnvironment(),
	)

	// If an application could not be created then err will reveal why.
	if err != nil {
		fmt.Println("unable to create New Relic Application", err)
	}

	return *app
}
