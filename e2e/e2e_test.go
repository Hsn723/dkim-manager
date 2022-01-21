package e2e

import (
	_ "embed"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	//go:embed t/selector1_dkimkey.yaml
	selector1YAML []byte

	//go:embed t/selector2_dkimkey.yaml
	selector2YAML []byte
)

var _ = Describe("dkim-manager", func() {
	It("should generate resources", func() {
		namespace := uuid.NewString()

		By("creating namespace")
		kubectlSafe(nil, "create", "ns", namespace)

		By("creating DKIMKey")
		kubectlSafe(selector1YAML, "apply", "-n", namespace, "-f", "-")

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "dnsendpoint", "selector1")
			return err
		}).Should(Succeed())

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "secret", "selector1")
			return err
		}).Should(Succeed())
	})

	It("should delete resources", func() {
		namespace := uuid.NewString()

		By("creating namespace")
		kubectlSafe(nil, "create", "ns", namespace)

		By("creating DKIMKey")
		kubectlSafe(selector1YAML, "apply", "-n", namespace, "-f", "-")

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "dnsendpoint", "selector1")
			return err
		}).Should(Succeed())

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "secret", "selector1")
			return err
		}).Should(Succeed())

		By("deleting DKIMKey")
		kubectlSafe(nil, "delete", "-n", namespace, "dkimkey", "selector1")

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "dnsendpoint", "selector1")
			return err
		}).ShouldNot(Succeed())

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "secret", "selector1")
			return err
		}).ShouldNot(Succeed())
	})

	It("should prevent user deletion of generated resources", func() {
		namespace := uuid.NewString()

		By("creating namespace")
		kubectlSafe(nil, "create", "ns", namespace)

		By("creating DKIMKey")
		kubectlSafe(selector1YAML, "apply", "-n", namespace, "-f", "-")

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "dnsendpoint", "selector1")
			return err
		}).Should(Succeed())

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "secret", "selector1")
			return err
		}).Should(Succeed())

		By("deleting DNSEndpoint")
		_, err := kubectl(nil, "delete", "-n", namespace, "dnsendpoint", "selector1")
		Expect(err).To(HaveOccurred())

		By("deleting Secret")
		_, err = kubectl(nil, "delete", "-n", namespace, "secret", "selector1")
		Expect(err).To(HaveOccurred())
	})

	It("should not delete resources owned by another DKIMKey", func() {
		namespace := uuid.NewString()

		By("creating namespace")
		kubectlSafe(nil, "create", "ns", namespace)

		By("creating DKIMKeys")
		kubectlSafe(selector1YAML, "apply", "-n", namespace, "-f", "-")
		kubectlSafe(selector2YAML, "apply", "-n", namespace, "-f", "-")

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "dnsendpoint", "selector1")
			return err
		}).Should(Succeed())

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "secret", "selector1")
			return err
		}).Should(Succeed())

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "dnsendpoint", "selector2")
			return err
		}).Should(Succeed())

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "secret", "selector2")
			return err
		}).Should(Succeed())

		By("deleting selector1 DKIMKey")
		kubectlSafe(nil, "delete", "-n", namespace, "dkimkey", "selector1")

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "dnsendpoint", "selector1")
			return err
		}).ShouldNot(Succeed())

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "secret", "selector1")
			return err
		}).ShouldNot(Succeed())

		Consistently(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "dnsendpoint", "selector2")
			return err
		}).Should(Succeed())

		Consistently(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "secret", "selector2")
			return err
		}).Should(Succeed())
	})
})
