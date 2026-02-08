package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	dkimmanagerv2 "github.com/hsn723/dkim-manager/api/v2"
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

func shouldCreateNamespace(ctx context.Context, namespace string) {
	By("creating namespace")
	err := k8sClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: namespace},
	})
	Expect(err).NotTo(HaveOccurred())
}

var _ = Describe("DKIMKey controller", func() {
	ctx := context.Background()
	var stopFunc func()

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:         scheme,
			LeaderElection: false,
			Metrics:        metricsserver.Options{BindAddress: "0"},
			Controller: config.Controller{
				SkipNameValidation: ptr.To(true),
			},
		})
		Expect(err).NotTo(HaveOccurred())
		reconciler := &DKIMKeyReconciler{
			Client:     mgr.GetClient(),
			Scheme:     mgr.GetScheme(),
			Log:        ctrl.Log.WithName("controllers").WithName("DKIMKey"),
			ReadClient: mgr.GetAPIReader(),
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
		shouldCreateNamespace(ctx, namespace)

		By("creating DKIMKey")
		dk := &dkimmanagerv2.DKIMKey{}
		dk.SetName(name)
		dk.SetNamespace(namespace)
		dk.Spec = dkimmanagerv2.DKIMKeySpec{
			SecretName: name,
			Selector:   "selector1",
			Domain:     "atelierhsn.com",
			TTL:        3600,
			KeyLength:  dkim.KeyLength2048,
			KeyType:    dkim.KeyTypeRSA,
		}

		err := k8sClient.Create(ctx, dk)
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

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(dk), dk)
		Expect(err).NotTo(HaveOccurred())
		Expect(dk.IsReady()).To(BeTrue())
	})

	It("should give up early when DNSEndpoint already exists", func() {
		name := uuid.NewString()
		namespace := uuid.NewString()
		shouldCreateNamespace(ctx, namespace)

		By("creating dummy DNSEndpoint")
		de := externaldns.DNSEndpoint()
		de.SetName(name)
		de.SetNamespace(namespace)
		dummyData := map[string]interface{}{
			"endpoints": []map[string]interface{}{
				{
					"dnsName":    "dummy",
					"recordType": "TXT",
					"targets":    []string{"dummy"},
				},
			},
		}
		de.UnstructuredContent()["spec"] = dummyData
		err := k8sClient.Create(ctx, de)
		Expect(err).NotTo(HaveOccurred())

		By("creating DKIMKey")
		dk := &dkimmanagerv2.DKIMKey{}
		dk.SetName(name)
		dk.SetNamespace(namespace)
		dk.Spec = dkimmanagerv2.DKIMKeySpec{
			SecretName: name,
			Selector:   "selector1",
			Domain:     "atelierhsn.com",
			TTL:        3600,
			KeyLength:  dkim.KeyLength2048,
			KeyType:    dkim.KeyTypeRSA,
		}

		err = k8sClient.Create(ctx, dk)
		Expect(err).NotTo(HaveOccurred())
		Consistently(func() error {
			return getSecret(ctx, name, namespace)
		}).ShouldNot(Succeed())
		Consistently(func() error {
			cde := externaldns.DNSEndpoint()
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(de), cde)
			if err != nil {
				return err
			}
			if !equality.Semantic.DeepEqual(cde.UnstructuredContent()["spec"], de.UnstructuredContent()["spec"]) {
				return fmt.Errorf("data has been modified. got: %v, expected %v", cde.UnstructuredContent()["spec"], de.UnstructuredContent()["spec"])
			}
			return nil
		}).Should(Succeed())

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(dk), dk)
		Expect(err).NotTo(HaveOccurred())
		Expect(dk.IsReady()).To(BeFalse())
	})

	It("should give up early when Secret already exists", func() {
		name := uuid.NewString()
		namespace := uuid.NewString()
		shouldCreateNamespace(ctx, namespace)

		By("creating dummy Secret")
		s := &corev1.Secret{}
		s.SetName(name)
		s.SetNamespace(namespace)
		dummyData := map[string][]byte{
			"dummy": []byte("dummy"),
		}
		s.Data = dummyData
		err := k8sClient.Create(ctx, s)
		Expect(err).NotTo(HaveOccurred())

		By("creating DKIMKey")
		dk := &dkimmanagerv2.DKIMKey{}
		dk.SetName(name)
		dk.SetNamespace(namespace)
		dk.Spec = dkimmanagerv2.DKIMKeySpec{
			SecretName: name,
			Selector:   "selector1",
			Domain:     "atelierhsn.com",
			TTL:        3600,
			KeyLength:  dkim.KeyLength2048,
			KeyType:    dkim.KeyTypeRSA,
		}

		err = k8sClient.Create(ctx, dk)
		Expect(err).NotTo(HaveOccurred())
		Consistently(func() error {
			return getDNSEndpoint(ctx, name, namespace)
		}).ShouldNot(Succeed())
		Consistently(func() error {
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(s), s)
			if err != nil {
				return err
			}
			if !equality.Semantic.DeepEqual(s.Data, dummyData) {
				return fmt.Errorf("data has been modified")
			}
			return nil
		}).Should(Succeed())

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(dk), dk)
		Expect(err).NotTo(HaveOccurred())
		Expect(dk.IsReady()).To(BeFalse())
	})

	It("should cascade delete generated resources", func() {
		name := uuid.NewString()
		namespace := uuid.NewString()
		userSecret := uuid.NewString()
		err := k8sClient.Create(ctx, &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{Name: namespace},
		})
		Expect(err).NotTo(HaveOccurred())

		By("creating DKIMKey")
		dk := &dkimmanagerv2.DKIMKey{}
		dk.SetName(name)
		dk.SetNamespace(namespace)
		dk.Spec = dkimmanagerv2.DKIMKeySpec{
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

		By("creating DKIMKeys")
		dk := &dkimmanagerv2.DKIMKey{}
		dk.SetName(dk1Name)
		dk.SetNamespace(namespace)
		dk.Spec = dkimmanagerv2.DKIMKeySpec{
			SecretName: dk1Name,
			Selector:   "selector1",
			Domain:     "atelierhsn.com",
			TTL:        3600,
			KeyType:    dkim.KeyTypeED25519,
		}

		dk2 := &dkimmanagerv2.DKIMKey{}
		dk2.SetName(dk2Name)
		dk2.SetNamespace(namespace)
		dk2.Spec = dkimmanagerv2.DKIMKeySpec{
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

		By("deleting DKIMKey")
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

var _ = Describe("DKIMKey controller namespaced", func() {
	ctx := context.Background()
	var stopFunc func()
	observedNamespace := uuid.NewString()

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:         scheme,
			LeaderElection: false,
			Metrics:        metricsserver.Options{BindAddress: "0"},
			Controller: config.Controller{
				SkipNameValidation: ptr.To(true),
			},
		})
		Expect(err).NotTo(HaveOccurred())
		reconciler := &DKIMKeyReconciler{
			Client:     mgr.GetClient(),
			Scheme:     mgr.GetScheme(),
			Log:        ctrl.Log.WithName("controllers").WithName("DKIMKey"),
			Namespace:  observedNamespace,
			ReadClient: mgr.GetAPIReader(),
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

	It("should reconcile resources in watched namespace", func() {
		name := uuid.NewString()
		namespace := observedNamespace
		err := k8sClient.Create(ctx, &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{Name: namespace},
		})
		Expect(err).NotTo(HaveOccurred())

		By("creating DKIMKey")
		dk := &dkimmanagerv2.DKIMKey{}
		dk.SetName(name)
		dk.SetNamespace(namespace)
		dk.Spec = dkimmanagerv2.DKIMKeySpec{
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

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(dk), dk)
		Expect(err).NotTo(HaveOccurred())
		Expect(dk.IsReady()).To(BeTrue())
	})

	It("should ignore resources in invalid namespaces", func() {
		name := uuid.NewString()
		namespace := uuid.NewString()
		err := k8sClient.Create(ctx, &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{Name: namespace},
		})
		Expect(err).NotTo(HaveOccurred())

		By("creating DKIMKey")
		dk := &dkimmanagerv2.DKIMKey{}
		dk.SetName(name)
		dk.SetNamespace(namespace)
		dk.Spec = dkimmanagerv2.DKIMKeySpec{
			SecretName: name,
			Selector:   "selector1",
			Domain:     "atelierhsn.com",
			TTL:        3600,
			KeyLength:  dkim.KeyLength2048,
			KeyType:    dkim.KeyTypeRSA,
		}

		err = k8sClient.Create(ctx, dk)
		Expect(err).NotTo(HaveOccurred())

		Consistently(func() error {
			return getSecret(ctx, name, namespace)
		}).ShouldNot(Succeed())

		Consistently(func() error {
			return getDNSEndpoint(ctx, name, namespace)
		}).ShouldNot(Succeed())

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(dk), dk)
		Expect(err).NotTo(HaveOccurred())
		Expect(dk.IsReady()).To(BeFalse())
	})
})
