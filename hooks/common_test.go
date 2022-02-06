package hooks

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dkimmanagerv1 "github.com/hsn723/dkim-manager/api/v1"
	"github.com/hsn723/dkim-manager/pkg/dkim"
)

func dummyDKIMKeySpec(name string) dkimmanagerv1.DKIMKeySpec {
	return dkimmanagerv1.DKIMKeySpec{
		SecretName: name,
		Selector:   "selector1",
		Domain:     "atelierhsn.com",
		TTL:        3600,
		KeyLength:  dkim.KeyLength2048,
		KeyType:    dkim.KeyTypeRSA,
	}
}

func shouldCreateNamespace(ctx context.Context, namespace string) {
	By("creating namespace")
	err := k8sClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: namespace},
	})
	Expect(err).NotTo(HaveOccurred())
}

func shouldCreateDKIMKey(ctx context.Context, name, namespace string, spec dkimmanagerv1.DKIMKeySpec) {
	By("creating DKIMKey")
	dk := &dkimmanagerv1.DKIMKey{}
	dk.SetName(name)
	dk.SetNamespace(namespace)
	dk.Spec = spec

	err := k8sClient.Create(ctx, dk)
	Expect(err).NotTo(HaveOccurred())
}
