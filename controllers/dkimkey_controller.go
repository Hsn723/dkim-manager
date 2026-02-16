/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	dkimmanagerv2 "github.com/hsn723/dkim-manager/api/v2"
	"github.com/hsn723/dkim-manager/pkg/dkim"
	"github.com/hsn723/dkim-manager/pkg/externaldns"
)

const (
	finalizerName = "dkim-manager.atelierhsn.com/finalizer"
	apiGroup      = "dkim-manager.atelierhsn.com"

	fieldOwner client.FieldOwner = "dkim-manager"
)

// DKIMKeyReconciler reconciles a DKIMKey object.
type DKIMKeyReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Namespace string
	// workaround for https://github.com/kubernetes-sigs/controller-runtime/issues/550
	ReadClient client.Reader
}

//+kubebuilder:rbac:groups=dkim-manager.atelierhsn.com,resources=dkimkeys,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dkim-manager.atelierhsn.com,resources=dkimkeys/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dkim-manager.atelierhsn.com,resources=dkimkeys/finalizers,verbs=update
//+kubebuilder:rbac:groups=externaldns.k8s.io,resources=dnsendpoints,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *DKIMKeyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	dk := &dkimmanagerv2.DKIMKey{}
	if err := r.Get(ctx, req.NamespacedName, dk); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if r.Namespace != "" && dk.Namespace != r.Namespace {
		if r.hasCondition(dk, dkimmanagerv2.ConditionReady, v1.ConditionFalse, dkimmanagerv2.ReasonInvalid) {
			return ctrl.Result{}, nil
		}
		logger.Info("dkimkey is in an invalid namespace, ignoring")
		r.setCondition(dk, dkimmanagerv2.ConditionReady, v1.ConditionFalse, dkimmanagerv2.ReasonInvalid, "DKIMKey is in an invalid namespace")
		return ctrl.Result{}, r.Status().Update(ctx, dk)
	}

	if dk.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(dk, finalizerName) {
			controllerutil.AddFinalizer(dk, finalizerName)
			return ctrl.Result{}, r.Update(ctx, dk)
		}
	} else {
		logger.Info("finalizing")
		return ctrl.Result{}, r.finalize(ctx, dk)
	}

	if dk.IsReady() && dk.Status.ObservedGeneration == dk.Generation {
		return ctrl.Result{}, nil
	}

	return r.reconcile(ctx, dk)
}

// setCondition updates the status condition on the DKIMKey.
func (r *DKIMKeyReconciler) setCondition(dk *dkimmanagerv2.DKIMKey, condType string, status v1.ConditionStatus, reason, message string) {
	dk.Status.ObservedGeneration = dk.Generation
	meta.SetStatusCondition(&dk.Status.Conditions, v1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: dk.Generation,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: v1.Now(),
	})
}

// hasCondition checks if the DKIMKey has a specific condition.
func (r *DKIMKeyReconciler) hasCondition(dk *dkimmanagerv2.DKIMKey, condType string, status v1.ConditionStatus, reason string) bool {
	for _, c := range dk.Status.Conditions {
		if c.Type == condType && c.Status == status && c.Reason == reason {
			return true
		}
	}
	return false
}

func (r *DKIMKeyReconciler) isOwnedByDKIMKey(dk *dkimmanagerv2.DKIMKey, ownerRefs []v1.OwnerReference) bool {
	for _, owner := range ownerRefs {
		if owner.Kind == dkimmanagerv2.DKIMKeyKind && owner.Name == dk.Name {
			gv, err := schema.ParseGroupVersion(owner.APIVersion)
			if err != nil {
				continue
			}
			if gv.Group == apiGroup {
				return true
			}
		}
	}
	return false
}

func (r *DKIMKeyReconciler) finalize(ctx context.Context, dk *dkimmanagerv2.DKIMKey) error {
	if !controllerutil.ContainsFinalizer(dk, finalizerName) {
		return nil
	}
	logger := log.FromContext(ctx)
	del := externaldns.DNSEndpointList()
	lo := &client.ListOptions{Namespace: dk.Namespace}
	if err := r.ReadClient.List(ctx, del, lo); client.IgnoreNotFound(err) != nil {
		return err
	}
	for _, de := range del.Items {
		if !r.isOwnedByDKIMKey(dk, de.GetOwnerReferences()) {
			continue
		}
		if err := r.Delete(ctx, &de); err != nil {
			return err
		}
	}
	ss := &corev1.SecretList{}
	if err := r.ReadClient.List(ctx, ss, lo); err != nil {
		return err
	}
	for _, s := range ss.Items {
		if !r.isOwnedByDKIMKey(dk, s.GetOwnerReferences()) {
			continue
		}
		if err := r.Delete(ctx, &s); err != nil {
			return err
		}
	}
	logger.Info("done finalizing")
	controllerutil.RemoveFinalizer(dk, finalizerName)
	return r.Update(ctx, dk)
}

func (r *DKIMKeyReconciler) reconcile(ctx context.Context, dk *dkimmanagerv2.DKIMKey) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var key []byte
	var pub string
	var reason string
	var err error
	var targets []string
	targets, err = r.checkForExistingKey(ctx, dk)
	if err != nil {
		logger.Error(err, "precondition failed")
		r.setCondition(dk, dkimmanagerv2.ConditionReady, v1.ConditionFalse, dkimmanagerv2.ReasonInvalid, err.Error())
		return ctrl.Result{}, r.Status().Update(ctx, dk)
	}
	if targets == nil {
		logger.Info("generating new key pair", "selector", dk.Spec.Selector)
		key, pub, reason, err = r.generateKeyPair(dk)
		if err != nil {
			logger.Error(err, "failed to generate key pair")
			r.setCondition(dk, dkimmanagerv2.ConditionReady, v1.ConditionFalse, reason, err.Error())
			return ctrl.Result{}, r.Status().Update(ctx, dk)
		}
		if err := r.reconcileDKIMPrivateKey(ctx, dk, key); err != nil {
			logger.Error(err, "failed to reconcile Secret")
			r.setCondition(dk, dkimmanagerv2.ConditionReady, v1.ConditionFalse, dkimmanagerv2.ReasonFailed, fmt.Sprintf("Failed to reconcile Secret: %v", err))
			return ctrl.Result{}, r.Status().Update(ctx, dk)
		}
		targets = []string{dkim.GenTXTValue(pub, dk.Spec.KeyType)}
	}
	if err := r.reconcileDKIMRecord(ctx, dk, targets); err != nil {
		logger.Error(err, "failed to reconcile DNSEndpoint")
		r.setCondition(dk, dkimmanagerv2.ConditionReady, v1.ConditionFalse, dkimmanagerv2.ReasonFailed, fmt.Sprintf("Failed to reconcile DNSEndpoint: %v", err))
		return ctrl.Result{}, r.Status().Update(ctx, dk)
	}
	logger.Info("done reconciling DKIMKey")
	r.setCondition(dk, dkimmanagerv2.ConditionReady, v1.ConditionTrue, dkimmanagerv2.ReasonSucceeded, "DKIM key created successfully")
	return ctrl.Result{}, r.Status().Update(ctx, dk)
}

func (r DKIMKeyReconciler) generateKeyPair(dk *dkimmanagerv2.DKIMKey) (key []byte, pub, reason string, err error) {
	switch dk.Spec.KeyType {
	case dkim.KeyTypeRSA:
		key, pub, err = dkim.GenRSA(dk.Spec.KeyLength)
	case dkim.KeyTypeED25519:
		key, pub, err = dkim.GenED25519()
	default:
		reason = dkimmanagerv2.ReasonInvalid
		err = fmt.Errorf("invalid key type specified")
		return
	}
	if err != nil {
		reason = dkimmanagerv2.ReasonFailed
		err = fmt.Errorf("failed to generate key: %v", err)
		return
	}
	return
}

func (r *DKIMKeyReconciler) checkForExistingKey(ctx context.Context, dk *dkimmanagerv2.DKIMKey) ([]string, error) {
	var targets []string
	// If the private key does not exist, subsequent checks can be short-circuited.
	// Otherwise, the public key can be derived from the private key.
	s := &corev1.Secret{}
	sKey := client.ObjectKey{
		Namespace: dk.Namespace,
		Name:      dk.Spec.SecretName,
	}
	err := r.ReadClient.Get(ctx, sKey, s)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to check for existing Secret: %v", err)
	}
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	priv, ok := s.Data[r.generatePrivateKeyFilename(dk)]
	if !ok {
		return nil, fmt.Errorf("private key not found in Secret")
	}
	var pub string
	switch dk.Spec.KeyType {
	case dkim.KeyTypeRSA:
		pub, err = dkim.DeriveRSAPublicKey(priv, dk.Spec.KeyLength)
	case dkim.KeyTypeED25519:
		pub, err = dkim.DeriveED25519PublicKey(priv)
	default:
		return nil, fmt.Errorf("invalid key type specified")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to derive public key: %v", err)
	}
	targets = []string{dkim.GenTXTValue(pub, dk.Spec.KeyType)}
	return targets, nil
}

func (r *DKIMKeyReconciler) reconcileDKIMRecord(ctx context.Context, dk *dkimmanagerv2.DKIMKey, targets []string) error {
	logger := log.FromContext(ctx)
	de := externaldns.DNSEndpoint()
	de.SetName(dk.Name)
	de.SetNamespace(dk.Namespace)
	de.UnstructuredContent()["spec"] = map[string]interface{}{
		"endpoints": []map[string]interface{}{
			{
				"dnsName":    fmt.Sprintf("%s._domainkey.%s", dk.Spec.Selector, dk.Spec.Domain),
				"recordTTL":  dk.Spec.TTL,
				"recordType": "TXT",
				"targets":    targets,
			},
		},
	}
	if err := ctrl.SetControllerReference(dk, de, r.Scheme); err != nil {
		return err
	}
	ac := client.ApplyConfigurationFromUnstructured(de)
	if err := r.Apply(ctx, ac, fieldOwner, client.ForceOwnership); err != nil {
		return err
	}
	logger.Info("done reconciling DNSEndpoint")
	return nil
}

func (r *DKIMKeyReconciler) reconcileDKIMPrivateKey(ctx context.Context, dk *dkimmanagerv2.DKIMKey, key []byte) error {
	logger := log.FromContext(ctx)
	filename := r.generatePrivateKeyFilename(dk)
	s := &corev1.Secret{}
	s.SetName(dk.Spec.SecretName)
	s.SetNamespace(dk.Namespace)
	s.Immutable = ptr.To(true)
	s.Data = map[string][]byte{
		filename: key,
	}
	if err := ctrl.SetControllerReference(dk, s, r.Scheme); err != nil {
		return err
	}
	if err := r.Create(ctx, s); err != nil {
		return err
	}
	logger.Info("done reconciling Secret")
	return nil
}

func (r DKIMKeyReconciler) generatePrivateKeyFilename(dk *dkimmanagerv2.DKIMKey) string {
	return fmt.Sprintf("%s.%s.key", dk.Spec.Domain, dk.Spec.Selector)
}

// SetupWithManager sets up the controller with the Manager.
func (r *DKIMKeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dkimmanagerv2.DKIMKey{}).
		Complete(r)
}
