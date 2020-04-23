// +build integration

package controllers

import (
	"context"
	"github.com/newrelic/newrelic-client-go/pkg/alerts"

	//"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nralertsv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces/interfacesfakes"

	//v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	//ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("policy reconciliation", func() {
	var (
		ctx     context.Context
		r       *PolicyReconciler
		policy  *nralertsv1.Policy
		request ctrl.Request
		namespacedName types.NamespacedName
		//expectedEvents []string
		//secret        *v1.Secret
		fakeAlertFunc func(string, string) (interfaces.NewRelicAlertsClient, error)
	)

	BeforeEach(func() {
		ctx = context.Background()

		alertsClient = &interfacesfakes.FakeNewRelicAlertsClient{}

		fakeAlertFunc = func(string, string) (interfaces.NewRelicAlertsClient, error) {
			return alertsClient, nil
		}

		alertsClient.CreatePolicyStub = func(a alerts.Policy) (*alerts.Policy, error) {
			a.ID = 333
			return &a, nil
		}

		r = &PolicyReconciler{
			Client:          k8sClient,
			Log:             logf.Log,
			AlertClientFunc: fakeAlertFunc,
		}

		policy = &nralertsv1.Policy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-policy",
				Namespace: "default",
			},
			Spec: nralertsv1.PolicySpec{
				Name: "test policy",
				APIKey: "112233",
			},
			Status: nralertsv1.PolicyStatus{
				AppliedSpec: &nralertsv1.PolicySpec{},
				PolicyID: 0,
			},
		}

		namespacedName = types.NamespacedName{
			Namespace: "default",
			Name:      "test-policy",
		}
		request = ctrl.Request{NamespacedName: namespacedName}


	})

	Context("When starting with no policies", func() {
		Context("with a valid policy", func() {
			It("should create that policy", func() {

				err := k8sClient.Create(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

			})

			It("updates the policyId on the Policy resource", func() {

				err := k8sClient.Create(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStatePolicy nralertsv1.Policy
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(err).To(BeNil())
				Expect(endStatePolicy.Status.PolicyID).To(Equal(333))

			})
		})

	})

	AfterEach(func() {
		// Delete the policy
		err := k8sClient.Delete(ctx, policy)
		Expect(err).ToNot(HaveOccurred())

		// Need to call reconcile to delete finalizer
		_, err = r.Reconcile(request)
		Expect(err).ToNot(HaveOccurred())
	})

})
