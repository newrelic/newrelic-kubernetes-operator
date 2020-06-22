package main

import (
	"fmt"
	"os"

	"github.com/newrelic/go-agent/v3/newrelic"
)

//InitializeNRAgent - Initializes the NR Agent
func InitializeNRAgent() *newrelic.Application {

	//Get License key from ENV var- NEW_RELIC_LICENSE_KEY

	licenseKey := os.Getenv("NEW_RELIC_LICENSE_KEY")
	// if licenseKey == "" {
	// 	return newrelic.
	// }

	// Get App name from NEW_RELIC_APP_NAME
	appName := os.Getenv("NEW_RELIC_APP_NAME")
	if appName == "" {
		appName = "New Relic Kubernetes Operator"
	}

	app, err := newrelic.NewApplication(
		// Name your application
		newrelic.ConfigAppName(appName),
		// Fill in your New Relic license key
		newrelic.ConfigLicense(licenseKey),
		// Add logging:
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	// If an application could not be created then err will reveal why.
	if err != nil {
		fmt.Println("unable to create New Relic Application", err)
	}

	return app
}
