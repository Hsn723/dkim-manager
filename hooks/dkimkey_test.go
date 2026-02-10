package hooks

import (
	"context"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dkimmanagerv1 "github.com/hsn723/dkim-manager/api/v1"
	dkimmanagerv2 "github.com/hsn723/dkim-manager/api/v2"
)

var _ = Describe("DKIMKey v1 webhook", func() {
	ctx := context.Background()

	It("should allow adding labels", func() {
		name := uuid.NewString()
		namespace := uuid.NewString()
		shouldCreateNamespace(ctx, namespace)
		shouldCreateV1DKIMKey(ctx, name, namespace, dummyV1DKIMKeySpec(name))

		dk := &dkimmanagerv1.DKIMKey{}
		key := client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}
		By("adding labels")
		err := k8sClient.Get(ctx, key, dk)
		Expect(err).NotTo(HaveOccurred())

		dk.SetLabels(map[string]string{
			"test": "label",
		})

		err = k8sClient.Update(ctx, dk)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should deny changing name", func() {
		name := uuid.NewString()
		namespace := uuid.NewString()
		shouldCreateNamespace(ctx, namespace)
		shouldCreateV1DKIMKey(ctx, name, namespace, dummyV1DKIMKeySpec(name))

		dk := &dkimmanagerv1.DKIMKey{}
		key := client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}

		By("changing name")
		err := k8sClient.Get(ctx, key, dk)
		Expect(err).NotTo(HaveOccurred())
		dk.SetName(uuid.NewString())

		err = k8sClient.Update(ctx, dk)
		Expect(err).To(HaveOccurred())
	})

	It("should deny changing spec", func() {
		name := uuid.NewString()
		namespace := uuid.NewString()
		shouldCreateNamespace(ctx, namespace)
		shouldCreateV1DKIMKey(ctx, name, namespace, dummyV1DKIMKeySpec(name))

		dk := &dkimmanagerv1.DKIMKey{}
		key := client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}

		By("changing spec")
		err := k8sClient.Get(ctx, key, dk)
		Expect(err).NotTo(HaveOccurred())
		dk.Spec.SecretName = uuid.NewString()

		err = k8sClient.Update(ctx, dk)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("DKIMKey v2 webhook", func() {
	ctx := context.Background()

	It("should allow adding labels", func() {
		name := uuid.NewString()
		namespace := uuid.NewString()
		shouldCreateNamespace(ctx, namespace)
		shouldCreateDKIMKey(ctx, name, namespace, dummyDKIMKeySpec(name))

		dk := &dkimmanagerv2.DKIMKey{}
		key := client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}
		By("adding labels")
		err := k8sClient.Get(ctx, key, dk)
		Expect(err).NotTo(HaveOccurred())

		dk.SetLabels(map[string]string{
			"test": "label",
		})

		err = k8sClient.Update(ctx, dk)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should deny changing name", func() {
		name := uuid.NewString()
		namespace := uuid.NewString()
		shouldCreateNamespace(ctx, namespace)
		shouldCreateDKIMKey(ctx, name, namespace, dummyDKIMKeySpec(name))

		dk := &dkimmanagerv2.DKIMKey{}
		key := client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}

		By("changing name")
		err := k8sClient.Get(ctx, key, dk)
		Expect(err).NotTo(HaveOccurred())
		dk.SetName(uuid.NewString())

		err = k8sClient.Update(ctx, dk)
		Expect(err).To(HaveOccurred())
	})

	It("should deny changing spec", func() {
		name := uuid.NewString()
		namespace := uuid.NewString()
		shouldCreateNamespace(ctx, namespace)
		shouldCreateDKIMKey(ctx, name, namespace, dummyDKIMKeySpec(name))

		dk := &dkimmanagerv2.DKIMKey{}
		key := client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}

		By("changing spec")
		err := k8sClient.Get(ctx, key, dk)
		Expect(err).NotTo(HaveOccurred())
		dk.Spec.SecretName = uuid.NewString()

		err = k8sClient.Update(ctx, dk)
		Expect(err).To(HaveOccurred())
	})
})
