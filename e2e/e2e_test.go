package e2e

import (
	"crypto/rsa"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	//go:embed t/selector1_dkimkey.yaml
	selector1YAML []byte

	//go:embed t/selector2_dkimkey.yaml
	selector2YAML []byte

	pubKeyPattern = regexp.MustCompile(`(p=)(\S+)`)
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

		By("checking the key pair")
		Consistently(func() error {
			record, err := kubectl(nil, "get", "-n", namespace, "dnsendpoint", "selector1", "-o", "jsonpath={.spec.endpoints[*].targets[*]}")
			if err != nil {
				return err
			}
			match := pubKeyPattern.FindStringSubmatch(string(record))
			if len(match) != 3 {
				return fmt.Errorf("unexpected DKIM record")
			}
			pubKey := match[2]

			privKeyData, err := kubectl(nil, "get", "-n", namespace, "secret", "selector1", "-o", "go-template={{index .data \"example.com.selector1.key\"}}")
			if err != nil {
				return err
			}
			b := make([]byte, base64.StdEncoding.DecodedLen(len(privKeyData)))
			_, err = base64.StdEncoding.Decode(b, privKeyData)
			if err != nil {
				return err
			}
			block, _ := pem.Decode(b)
			privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return err
			}
			pubKeyBytes, err := x509.MarshalPKIXPublicKey(privKey.Public().(*rsa.PublicKey))
			if err != nil {
				return err
			}
			pubKeyStr := base64.StdEncoding.EncodeToString(pubKeyBytes)
			if pubKeyStr != pubKey {
				return fmt.Errorf("incorrect key pair")
			}
			return nil
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
		kubectlSafe(nil, "delete", "-n", namespace, "dkimkey", "selector1", "--cascade=foreground")

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
