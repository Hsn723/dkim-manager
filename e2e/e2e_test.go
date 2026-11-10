package e2e

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"uuid"

	"github.com/hsn723/dkim-manager/pkg/dkim"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	dkimKeyTemplate = `apiVersion: dkim-manager.atelierhsn.com/v2
kind: DKIMKey
metadata:
  name: {{ .Name }}
spec:
  secretName: {{ .Name }}
  selector: {{ .Name }}
  domain: example.com
  keyType: {{ .KeyType }}`
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
		namespace := uuid.New().String()

		By("creating namespace")
		kubectlSafe(nil, "create", "ns", namespace)

		dkimKeys := []struct {
			Name    string
			KeyType dkim.KeyType
		}{
			{
				Name:    "selector1",
				KeyType: dkim.KeyTypeRSA,
			},
			{
				Name:    "selector2",
				KeyType: dkim.KeyTypeED25519,
			},
			{
				Name:    "selector3",
				KeyType: dkim.KeyTypeMLDSA44,
			},
		}
		tmpl := template.Must(template.New("dkimKey").Parse(dkimKeyTemplate))

		By("creating DKIMKeys")
		for _, dk := range dkimKeys {
			var data bytes.Buffer
			err := tmpl.Execute(&data, dk)
			Expect(err).ToNot(HaveOccurred())
			kubectlSafe(data.Bytes(), "apply", "-n", namespace, "-f", "-")
		}
		checkDKIMKeyResources := func(name string) error {
			_, err := kubectl(nil, "get", "-n", namespace, "dnsendpoint", name)
			if err != nil {
				return err
			}
			_, err = kubectl(nil, "get", "-n", namespace, "secret", name)
			if err != nil {
				return err
			}
			return nil
		}
		Eventually(func() error {
			for _, dk := range dkimKeys {
				if err := checkDKIMKeyResources(dk.Name);err != nil {
					return err
				}
			}
			return nil
		}).Should(Succeed())

		By("checking the key pairs")
		checkKeyPair := func(name string, keyType dkim.KeyType) error {
			record, err := kubectl(nil, "get", "-n", namespace, "dnsendpoint", name, "-o", "jsonpath={.spec.endpoints[*].targets[*]}")
			if err != nil {
				return err
			}
			trimmedRecord := strings.ReplaceAll(strings.ReplaceAll(string(record), " ", ""), "\"", "")
			match := pubKeyPattern.FindStringSubmatch(string(trimmedRecord))
			if len(match) != 3 {
				return fmt.Errorf("unexpected DKIM record")
			}
			pubKey := match[2]

			privKeyData, err := kubectl(nil, "get", "-n", namespace, "secret", name, "-o", "go-template={{index .data \"example.com." + name + ".key\"}}")
			if err != nil {
				return err
			}
			privKey := make([]byte, base64.StdEncoding.DecodedLen(len(privKeyData)))
			_, err = base64.StdEncoding.Decode(privKey, privKeyData)
			dp, err := dkim.NewKeyProvider(keyType, dkim.KeyLength2048)
			if err != nil {
				return err
			}
			pub, err := dp.PublicKey(privKey)
			if err != nil {
				return err
			}
			if pub != pubKey {
				return fmt.Errorf("incorrect key pair for %s", name)
			}
			return nil
		}
		Consistently(func() error {
			for _, dk := range dkimKeys {
				if err := checkKeyPair(dk.Name, dk.KeyType); err != nil {
					return err
				}
			}
			return nil
		}).Should(Succeed())

		By("checking the DKIMKey statuses")
		checkDKIMKeyStatuses := func() error {
			for _, dk := range dkimKeys {
				status, err := kubectl(nil, "get", "-n", namespace, "dkimkey", dk.Name, "-o", "jsonpath={.status.conditions[?(@.type==\"Ready\")].status}")
				if err != nil {
					return err
				}
				if string(status) != "True" {
					return fmt.Errorf("DKIMKey is not ready")
				}
			}
			return nil
		}
		Eventually(checkDKIMKeyStatuses).Should(Succeed())
		Consistently(checkDKIMKeyStatuses).Should(Succeed())
	})

	It("should delete resources", func() {
		namespace := uuid.New().String()

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
		namespace := uuid.New().String()

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
		namespace := uuid.New().String()

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
