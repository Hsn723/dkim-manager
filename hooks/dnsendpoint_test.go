package hooks

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dkimmanagerv1 "github.com/hsn723/dkim-manager/api/v1"
	"github.com/hsn723/dkim-manager/pkg/externaldns"
)

var _ = Describe("DNSEndpoint webhook", func() {
	ctx := context.Background()

	It("should allow deleting ordinary DNSEndpoints", func() {
		name := uuid.NewString()
		namespace := uuid.NewString()
		shouldCreateNamespace(ctx, namespace)

		By("creating DNSEndpoint")
		de := externaldns.DNSEndpoint()
		de.SetName(name)
		de.SetNamespace(namespace)
		de.UnstructuredContent()["spec"] = map[string]interface{}{
			"endpoints": []map[string]interface{}{
				{
					"dnsName":    "hoge",
					"recordTTL":  3600,
					"recordType": "TXT",
					"targets":    []string{"hoge"},
				},
			},
		}

		err := k8sClient.Create(ctx, de)
		Expect(err).NotTo(HaveOccurred())

		By("deleting endpoint")
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(de), de)
		Expect(err).NotTo(HaveOccurred())

		err = k8sClient.Delete(ctx, de)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should prevent deleting owned DNSEndpoints", func() {
		name := uuid.NewString()
		namespace := uuid.NewString()
		shouldCreateNamespace(ctx, namespace)
		shouldCreateDKIMKey(ctx, name, namespace, dummyDKIMKeySpec(name))

		dk := &dkimmanagerv1.DKIMKey{}
		key := client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}
		err := k8sClient.Get(ctx, key, dk)
		Expect(err).NotTo(HaveOccurred())

		By("creating DNSEndpoint")
		de := externaldns.DNSEndpoint()
		de.SetName(name)
		de.SetNamespace(namespace)
		de.UnstructuredContent()["spec"] = map[string]interface{}{
			"endpoints": []map[string]interface{}{
				{
					"dnsName":    "hoge",
					"recordTTL":  3600,
					"recordType": "TXT",
					"targets":    []string{"hoge"},
				},
			},
		}
		de.SetOwnerReferences([]v1.OwnerReference{
			{
				APIVersion: dkimmanagerv1.GroupVersion.String(),
				Kind:       dkimmanagerv1.DKIMKeyKind,
				Name:       dk.GetName(),
				UID:        dk.GetUID(),
			},
		})

		err = k8sClient.Create(ctx, de)
		Expect(err).NotTo(HaveOccurred())

		By("deleting endpoint")
		err = k8sClient.Delete(ctx, de)
		Expect(err).To(HaveOccurred())
	})
})
