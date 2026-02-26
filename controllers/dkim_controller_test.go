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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	dkimmanagerv1 "github.com/hsn723/dkim-manager/api/v1"
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

	It("should allow existing DNSEndpoint with no public keys", func() {
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
		Eventually(func() error {
			return getSecret(ctx, name, namespace)
		}).Should(Succeed())
		Consistently(func() error {
			return getSecret(ctx, name, namespace)
		}).Should(Succeed())
		Eventually(func() error {
			cde := externaldns.DNSEndpoint()
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(de), cde)
			if err != nil {
				return err
			}
			endpoints, ok, err := unstructured.NestedSlice(cde.UnstructuredContent(), "spec", "endpoints")
			if err != nil || !ok || len(endpoints) == 0 {
				return fmt.Errorf("invalid endpoints: %v", err)
			}
			targets, ok := endpoints[0].(map[string]interface{})["targets"].([]interface{})
			if !ok {
				return fmt.Errorf("invalid targets")
			}
			if _, ok := targets[0].(string); !ok {
				return fmt.Errorf("invalid target value")
			}
			return nil
		}).Should(Succeed())

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(dk), dk)
		Expect(err).NotTo(HaveOccurred())
		Expect(dk.IsReady()).To(BeTrue())
	})

	It("should allow changing DKIMKey spec when DNSEndpoint already exists with the same public key", func() {
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

		Eventually(func() error {
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(dk), dk)
			if err != nil {
				return err
			}
			if !dk.IsReady() {
				return fmt.Errorf("DKIMKey is not ready")
			}
			return nil
		}).Should(Succeed())

		newTTL := uint(1800)
		dk.Spec.TTL = newTTL
		err = k8sClient.Update(ctx, dk)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() error {
			de := externaldns.DNSEndpoint()
			err := k8sClient.Get(ctx, client.ObjectKey{
				Name:      name,
				Namespace: namespace,
			}, de)
			if err != nil {
				return err
			}
			endpoints, ok, err := unstructured.NestedSlice(de.UnstructuredContent(), "spec", "endpoints")
			if err != nil || !ok || len(endpoints) == 0 {
				return fmt.Errorf("invalid endpoints: %v", err)
			}
			endpoint, ok := endpoints[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("endpoint not found")
			}
			ttl, ok := endpoint["recordTTL"].(int64)
			if !ok {
				return fmt.Errorf("invalid ttl type: %T", endpoint["recordTTL"])
			}
			if uint(ttl) != newTTL {
				return fmt.Errorf("unexpected ttl: got %d, want %d", ttl, newTTL)
			}
			return nil
		}).Should(Succeed())

		Eventually(k8sClient.Get).WithArguments(ctx, client.ObjectKeyFromObject(dk), dk).Should(Succeed())
		Expect(dk.IsReady()).To(BeTrue())
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
			Namespaces: []string{observedNamespace},
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

var _ = Describe("DKIMKey v1/v2 conversion", func() {
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

	It("should reconcile a DKIMKey created via v1 API", func() {
		name := uuid.NewString()
		namespace := uuid.NewString()
		shouldCreateNamespace(ctx, namespace)

		By("creating DKIMKey via v1 API")
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

		err := k8sClient.Create(ctx, dk)
		Expect(err).NotTo(HaveOccurred())

		By("verifying resources are created")
		Eventually(func() error {
			return getSecret(ctx, name, namespace)
		}).Should(Succeed())

		Eventually(func() error {
			return getDNSEndpoint(ctx, name, namespace)
		}).Should(Succeed())

		By("reading back via v1 and checking status")
		Eventually(func() error {
			v1Key := &dkimmanagerv1.DKIMKey{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(dk), v1Key)
			if err != nil {
				return err
			}
			if v1Key.Status != dkimmanagerv1.DKIMKeyStatusOK {
				return fmt.Errorf("unexpected status: got %s, want %s", v1Key.Status, dkimmanagerv1.DKIMKeyStatusOK)
			}
			return nil
		}).Should(Succeed())

		By("reading back via v2 and checking status")
		Eventually(func() error {
			v2Key := &dkimmanagerv2.DKIMKey{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(dk), v2Key)
			if err != nil {
				return err
			}
			if !v2Key.IsReady() {
				return fmt.Errorf("DKIMKey is not ready")
			}
			return nil
		}).Should(Succeed())
	})

	It("should show correct v1 status for a DKIMKey created via v2 API", func() {
		name := uuid.NewString()
		namespace := uuid.NewString()
		shouldCreateNamespace(ctx, namespace)

		By("creating DKIMKey via v2 API")
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

		By("reading back via v1 and checking status")
		Eventually(func() error {
			v1Key := &dkimmanagerv1.DKIMKey{}
			err = k8sClient.Get(ctx, client.ObjectKeyFromObject(dk), v1Key)
			if err != nil {
				return err
			}
			if v1Key.Status != dkimmanagerv1.DKIMKeyStatusOK {
				return fmt.Errorf("unexpected status: got %s, want %s", v1Key.Status, dkimmanagerv1.DKIMKeyStatusOK)
			}
			return nil
		}).Should(Succeed())
	})
})
