package controllers

import (
	"context"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dkimmanagerv1 "github.com/hsn723/dkim-manager/api/v1"
	"github.com/hsn723/dkim-manager/pkg/dkim"
	"github.com/hsn723/dkim-manager/pkg/externaldns"
)

func getDNSEndpoint(ctx context.Context, name, namespace string) error {
	de := externaldns.DNSEndpoint()
	key := client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}
	return k8sClient.Get(ctx, key, de)
}

func getSecret(ctx context.Context, name, namespace string) error {
	s := &corev1.Secret{}
	key := client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}
	return k8sClient.Get(ctx, key, s)
}

var _ = Describe("DKIMKey controller", func() {
	ctx := context.Background()
	var stopFunc func()

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:             scheme,
			LeaderElection:     false,
			MetricsBindAddress: "0",
		})
		Expect(err).NotTo(HaveOccurred())
		reconciler := &DKIMKeyReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
			Log:    ctrl.Log.WithName("controllers").WithName("DKIMKey"),
		}
		err = reconciler.SetupWithManager(mgr)
		Expect(err).NotTo(HaveOccurred())

		ctx, cancel := context.WithCancel(ctx)
		stopFunc = cancel
		go func() {
			err := mgr.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
		time.Sleep(100 * time.Millisecond)
	})

	AfterEach(func() {
		stopFunc()
		time.Sleep(100 * time.Millisecond)
	})

	It("should create DNSEndpoint and Secret", func() {
		name := uuid.NewString()
		namespace := uuid.NewString()
		err := k8sClient.Create(ctx, &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{Name: namespace},
		})
		Expect(err).NotTo(HaveOccurred())

		dk := &dkimmanagerv1.DKIMKey{}
		dk.SetName(name)
		dk.SetNamespace(namespace)
		dk.Spec = dkimmanagerv1.DKIMKeySpec{
			SecretName: name,
			Selector:   "selector1",
			Domain:     "atelierhsn.com",
			TTL:        3600,
			KeyLength:  dkim.KeyLength2048,
			KeyType:    dkim.KeyTypeRSA,
		}

		err = k8sClient.Create(ctx, dk)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() error {
			return getSecret(ctx, name, namespace)
		}).Should(Succeed())

		Eventually(func() error {
			return getDNSEndpoint(ctx, name, namespace)
		}).Should(Succeed())

		Consistently(func() error {
			return getSecret(ctx, name, namespace)
		}).Should(Succeed())

		Consistently(func() error {
			return getDNSEndpoint(ctx, name, namespace)
		}).Should(Succeed())
	})

	It("should cascade delete generated resources", func() {
		name := uuid.NewString()
		namespace := uuid.NewString()
		userSecret := uuid.NewString()
		err := k8sClient.Create(ctx, &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{Name: namespace},
		})
		Expect(err).NotTo(HaveOccurred())

		dk := &dkimmanagerv1.DKIMKey{}
		dk.SetName(name)
		dk.SetNamespace(namespace)
		dk.Spec = dkimmanagerv1.DKIMKeySpec{
			SecretName: name,
			Selector:   "selector1",
			Domain:     "atelierhsn.com",
			TTL:        3600,
			KeyType:    dkim.KeyTypeED25519,
		}

		err = k8sClient.Create(ctx, dk)
		Expect(err).NotTo(HaveOccurred())

		s := &corev1.Secret{}
		s.Name = userSecret
		s.Namespace = namespace
		s.StringData = map[string]string{
			"hoge": "hoge",
		}
		err = k8sClient.Create(ctx, s)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() error {
			return getSecret(ctx, name, namespace)
		}).Should(Succeed())

		Eventually(func() error {
			return getDNSEndpoint(ctx, name, namespace)
		}).Should(Succeed())

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(dk), dk)
		Expect(err).NotTo(HaveOccurred())

		err = k8sClient.Delete(ctx, dk)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() error {
			return getSecret(ctx, name, namespace)
		}).ShouldNot(Succeed())

		Eventually(func() error {
			return getDNSEndpoint(ctx, name, namespace)
		}).ShouldNot(Succeed())

		Consistently(func() error {
			return getSecret(ctx, userSecret, namespace)
		}).Should(Succeed())
	})

	It("should cascade delete owned generated resources", func() {
		dk1Name := uuid.NewString()
		dk2Name := uuid.NewString()
		namespace := uuid.NewString()
		err := k8sClient.Create(ctx, &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{Name: namespace},
		})
		Expect(err).NotTo(HaveOccurred())

		dk := &dkimmanagerv1.DKIMKey{}
		dk.SetName(dk1Name)
		dk.SetNamespace(namespace)
		dk.Spec = dkimmanagerv1.DKIMKeySpec{
			SecretName: dk1Name,
			Selector:   "selector1",
			Domain:     "atelierhsn.com",
			TTL:        3600,
			KeyType:    dkim.KeyTypeED25519,
		}

		dk2 := &dkimmanagerv1.DKIMKey{}
		dk2.SetName(dk2Name)
		dk2.SetNamespace(namespace)
		dk2.Spec = dkimmanagerv1.DKIMKeySpec{
			SecretName: dk2Name,
			Selector:   "selector2",
			Domain:     "atelierhsn.com",
			TTL:        3600,
			KeyType:    dkim.KeyTypeRSA,
		}

		err = k8sClient.Create(ctx, dk)
		Expect(err).NotTo(HaveOccurred())

		err = k8sClient.Create(ctx, dk2)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() error {
			if err := getSecret(ctx, dk1Name, namespace); err != nil {
				return err
			}
			if err := getSecret(ctx, dk2Name, namespace); err != nil {
				return err
			}
			if err := getDNSEndpoint(ctx, dk1Name, namespace); err != nil {
				return err
			}
			if err := getDNSEndpoint(ctx, dk2Name, namespace); err != nil {
				return err
			}
			return nil
		}).Should(Succeed())

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(dk), dk)
		Expect(err).NotTo(HaveOccurred())

		err = k8sClient.Delete(ctx, dk)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() error {
			return getSecret(ctx, dk1Name, namespace)
		}).ShouldNot(Succeed())

		Eventually(func() error {
			return getDNSEndpoint(ctx, dk1Name, namespace)
		}).ShouldNot(Succeed())

		Consistently(func() error {
			return getSecret(ctx, dk2Name, namespace)
		}).Should(Succeed())

		Consistently(func() error {
			return getDNSEndpoint(ctx, dk2Name, namespace)
		}).Should(Succeed())
	})
})
