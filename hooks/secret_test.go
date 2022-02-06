package hooks

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dkimmanagerv1 "github.com/hsn723/dkim-manager/api/v1"
)

var _ = Describe("Secret webhook", func() {
	ctx := context.Background()

	It("should allow deleting ordinary Secrets", func() {
		name := uuid.NewString()
		namespace := uuid.NewString()
		shouldCreateNamespace(ctx, namespace)

		By("creating Secret")
		s := &corev1.Secret{}
		s.SetName(name)
		s.SetNamespace(namespace)
		s.Data = map[string][]byte{
			"dummy": []byte("secret"),
		}

		err := k8sClient.Create(ctx, s)
		Expect(err).NotTo(HaveOccurred())

		By("deleting secret")
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(s), s)
		Expect(err).NotTo(HaveOccurred())

		err = k8sClient.Delete(ctx, s)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should prevent deleting owned Secrets", func() {
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

		By("creating Secret")
		s := &corev1.Secret{}
		s.SetName(name)
		s.SetNamespace(namespace)
		s.Data = map[string][]byte{
			"dummy": []byte("secret"),
		}
		s.SetOwnerReferences([]v1.OwnerReference{
			{
				APIVersion: dkimmanagerv1.GroupVersion.String(),
				Kind:       dkimmanagerv1.DKIMKeyKind,
				Name:       dk.GetName(),
				UID:        dk.GetUID(),
			},
		})

		err = k8sClient.Create(ctx, s)
		Expect(err).NotTo(HaveOccurred())

		By("deleting secret")
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(s), s)
		Expect(err).NotTo(HaveOccurred())

		err = k8sClient.Delete(ctx, s)
		Expect(err).To(HaveOccurred())
	})
})
