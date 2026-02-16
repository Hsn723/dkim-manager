package hooks

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dkimmanagerv1 "github.com/hsn723/dkim-manager/api/v1"
	dkimmanagerv2 "github.com/hsn723/dkim-manager/api/v2"
	"github.com/hsn723/dkim-manager/pkg/dkim"
)

type dkimKeyMutator[T any] func(dk *T)

var _ = Describe("DKIMKey v1 webhook", func() {
	ctx := context.Background()
	cases := []struct {
		accept  bool
		mutator dkimKeyMutator[dkimmanagerv1.DKIMKey]
		title   string
	}{
		{
			title:  "should allow adding labels",
			accept: true,
			mutator: func(dk *dkimmanagerv1.DKIMKey) {
				By("adding labels")
				dk.SetLabels(map[string]string{
					"test": "label",
				})
			},
		},
		{
			title: "should deny changing name",
			mutator: func(dk *dkimmanagerv1.DKIMKey) {
				By("changing name")
				dk.SetName(uuid.NewString())
			},
		},
		{
			title: "should deny changing secretName",
			mutator: func(dk *dkimmanagerv1.DKIMKey) {
				By("changing spec")
				dk.Spec.SecretName = uuid.NewString()
			},
		},
		{
			title: "should deny changing keyType",
			mutator: func(dk *dkimmanagerv1.DKIMKey) {
				By("changing spec")
				dk.Spec.KeyType = dkim.KeyTypeED25519
			},
		},
		{
			title: "should deny changing keyLength",
			mutator: func(dk *dkimmanagerv1.DKIMKey) {
				By("changing spec")
				dk.Spec.KeyLength = dkim.KeyLength1024
			},
		},
		{
			title: "should deny changing selector",
			mutator: func(dk *dkimmanagerv1.DKIMKey) {
				By("changing spec")
				dk.Spec.Selector = "selector2"
			},
		},
		{
			title: "should deny changing domain",
			mutator: func(dk *dkimmanagerv1.DKIMKey) {
				By("changing spec")
				dk.Spec.Domain = "example.com"
			},
		},
		{
			title:  "should allow changing TTL",
			accept: true,
			mutator: func(dk *dkimmanagerv1.DKIMKey) {
				By("changing spec")
				dk.Spec.TTL = 100
			},
		},
	}
	for _, c := range cases {
		It(c.title, func() {
			name := uuid.NewString()
			namespace := uuid.NewString()
			shouldCreateNamespace(ctx, namespace)
			shouldCreateV1DKIMKey(ctx, name, namespace, dummyV1DKIMKeySpec(name))

			dk := &dkimmanagerv1.DKIMKey{}
			key := client.ObjectKey{
				Namespace: namespace,
				Name:      name,
			}
			err := k8sClient.Get(ctx, key, dk)
			Expect(err).NotTo(HaveOccurred())

			c.mutator(dk)

			err = k8sClient.Update(ctx, dk)
			if c.accept {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
		})
	}
})

var _ = Describe("DKIMKey v2 webhook", func() {
	ctx := context.Background()

	cases := []struct {
		accept  bool
		mutator dkimKeyMutator[dkimmanagerv2.DKIMKey]
		title   string
	}{
		{
			title:  "should allow adding labels",
			accept: true,
			mutator: func(dk *dkimmanagerv2.DKIMKey) {
				By("adding labels")
				dk.SetLabels(map[string]string{
					"test": "label",
				})
			},
		},
		{
			title: "should deny changing name",
			mutator: func(dk *dkimmanagerv2.DKIMKey) {
				By("changing name")
				dk.SetName(uuid.NewString())
			},
		},
		{
			title: "should deny changing secretName",
			mutator: func(dk *dkimmanagerv2.DKIMKey) {
				By("changing spec")
				dk.Spec.SecretName = uuid.NewString()
			},
		},
		{
			title: "should deny changing keyType",
			mutator: func(dk *dkimmanagerv2.DKIMKey) {
				By("changing spec")
				dk.Spec.KeyType = dkim.KeyTypeED25519
			},
		},
		{
			title: "should deny changing keyLength",
			mutator: func(dk *dkimmanagerv2.DKIMKey) {
				By("changing spec")
				dk.Spec.KeyLength = dkim.KeyLength1024
			},
		},
		{
			title: "should deny changing selector",
			mutator: func(dk *dkimmanagerv2.DKIMKey) {
				By("changing spec")
				dk.Spec.Selector = "selector2"
			},
		},
		{
			title: "should deny changing domain",
			mutator: func(dk *dkimmanagerv2.DKIMKey) {
				By("changing spec")
				dk.Spec.Domain = "example.com"
			},
		},
		{
			title:  "should allow changing TTL",
			accept: true,
			mutator: func(dk *dkimmanagerv2.DKIMKey) {
				By("changing spec")
				dk.Spec.TTL = 100
			},
		},
	}
	for _, c := range cases {
		It(c.title, func() {
			name := uuid.NewString()
			namespace := uuid.NewString()
			shouldCreateNamespace(ctx, namespace)
			shouldCreateDKIMKey(ctx, name, namespace, dummyDKIMKeySpec(name))

			dk := &dkimmanagerv2.DKIMKey{}
			key := client.ObjectKey{
				Namespace: namespace,
				Name:      name,
			}
			err := k8sClient.Get(ctx, key, dk)
			Expect(err).NotTo(HaveOccurred())

			c.mutator(dk)

			err = k8sClient.Update(ctx, dk)
			if c.accept {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
		})
	}
})
