// +build integration

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
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	"github.com/newrelic/newrelic-client-go/pkg/config"
	"github.com/newrelic/newrelic-client-go/pkg/region"
	nralertsv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{envtest.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "configs", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = nralertsv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func RandSeq(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	var letters = []rune("abcdefghijklmnopqrstuvwxyz")

	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)

}

const (
	LogLevel  = "debug"                                                     // LogLevel used in mock configs
	UserAgent = "newrelic/newrelic-kubernetes-operator (automated testing)" // UserAgent used in mock configs
)

// NewIntegrationTestConfig grabs environment vars for required fields or skips the test.
// returns a fully saturated configuration
func NewIntegrationTestConfig() config.Config {
	envPersonalAPIKey := os.Getenv("NEW_RELIC_API_KEY")
	envAdminAPIKey := os.Getenv("NEW_RELIC_ADMIN_API_KEY")
	envRegion := os.Getenv("NEW_RELIC_REGION")

	if envPersonalAPIKey == "" && envAdminAPIKey == "" {
		log.Fatal("acceptance testing requires NEW_RELIC_API_KEY and NEW_RELIC_ADMIN_API_KEY")
	}

	cfg := config.New()

	// Set some defaults
	cfg.LogLevel = LogLevel
	cfg.UserAgent = UserAgent

	cfg.PersonalAPIKey = envPersonalAPIKey
	cfg.AdminAPIKey = envAdminAPIKey

	if envRegion != "" {
		regName, err := region.Parse(envRegion)

		reg, err := region.Get(regName)
		if err != nil {
			log.Error(err)
		}

		err = cfg.SetRegion(reg)
		if err != nil {
			log.Error(err)
		}
	}

	return cfg
}

func newIntegrationTestClient() alerts.Alerts {
	tc := NewIntegrationTestConfig()

	return alerts.New(tc)
}
