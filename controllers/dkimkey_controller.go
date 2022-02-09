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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	dkimmanagerv1 "github.com/hsn723/dkim-manager/api/v1"
	"github.com/hsn723/dkim-manager/pkg/dkim"
	"github.com/hsn723/dkim-manager/pkg/externaldns"
)

const (
	finalizerName = "dkim-manager.atelierhsn.com/finalizer"
)

// DKIMKeyReconciler reconciles a DKIMKey object.
type DKIMKeyReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
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

	dk := &dkimmanagerv1.DKIMKey{}
	if err := r.Get(ctx, req.NamespacedName, dk); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
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

	if dk.Status == dkimmanagerv1.DKIMKeyStatusOK {
		return ctrl.Result{}, nil
	}

	return r.reconcile(ctx, dk)
}

func (r *DKIMKeyReconciler) isOwnedByDKIMKey(dk *dkimmanagerv1.DKIMKey, ownerRefs []v1.OwnerReference) bool {
	for _, owner := range ownerRefs {
		if owner.APIVersion == dkimmanagerv1.GroupVersion.String() && owner.Kind == dkimmanagerv1.DKIMKeyKind && owner.Name == dk.Name {
			return true
		}
	}
	return false
}

func (r *DKIMKeyReconciler) finalize(ctx context.Context, dk *dkimmanagerv1.DKIMKey) error {
	if !controllerutil.ContainsFinalizer(dk, finalizerName) {
		return nil
	}
	logger := log.FromContext(ctx)
	del := externaldns.DNSEndpointList()
	lo := &client.ListOptions{Namespace: dk.Namespace}
	if err := r.List(ctx, del, lo); client.IgnoreNotFound(err) != nil {
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
	if err := r.List(ctx, ss, lo); err != nil {
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

func (r *DKIMKeyReconciler) reconcile(ctx context.Context, dk *dkimmanagerv1.DKIMKey) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var key []byte
	var pub string
	var err error
	switch dk.Spec.KeyType {
	case dkim.KeyTypeRSA:
		key, pub, err = dkim.GenRSA(dk.Spec.KeyLength)
	case dkim.KeyTypeED25519:
		key, pub, err = dkim.GenED25519()
	default:
		return ctrl.Result{}, fmt.Errorf("invalid key type specified")
	}
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := r.reconcileDKIMRecord(ctx, dk, pub); err != nil {
		logger.Error(err, "failed to reconcile DNSEndpoint")
		return ctrl.Result{}, err
	}
	if err := r.reconcileDKIMPrivateKey(ctx, dk, key); err != nil {
		logger.Error(err, "failed to reconcile Secret")
		return ctrl.Result{}, err
	}
	dk.Status = dkimmanagerv1.DKIMKeyStatusOK
	if err := r.Update(ctx, dk); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *DKIMKeyReconciler) reconcileDKIMRecord(ctx context.Context, dk *dkimmanagerv1.DKIMKey, pub string) error {
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
				"targets":    []string{dkim.GenTXTValue(pub, dk.Spec.KeyType)},
			},
		},
	}
	if err := ctrl.SetControllerReference(dk, de, r.Scheme); err != nil {
		return err
	}
	if err := r.Patch(ctx, de, client.Apply, &client.PatchOptions{
		Force:        pointer.BoolPtr(true),
		FieldManager: "dkim-manager",
	}); err != nil {
		return err
	}
	logger.Info("done reconciling DNSEndpoint")
	return nil
}

func (r *DKIMKeyReconciler) reconcileDKIMPrivateKey(ctx context.Context, dk *dkimmanagerv1.DKIMKey, key []byte) error {
	logger := log.FromContext(ctx)
	filename := fmt.Sprintf("%s.%s.key", dk.Spec.Domain, dk.Spec.Selector)
	s := &corev1.Secret{}
	s.SetName(dk.Spec.SecretName)
	s.SetNamespace(dk.Namespace)
	s.Immutable = pointer.BoolPtr(true)
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

// SetupWithManager sets up the controller with the Manager.
func (r *DKIMKeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dkimmanagerv1.DKIMKey{}).
		Complete(r)
}
