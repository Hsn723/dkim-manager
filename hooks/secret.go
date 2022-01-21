package hooks

import (
	"context"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	dkimmanagerv1 "github.com/hsn723/dkim-manager/api/v1"
)

//+kubebuilder:webhook:path=/validate-secret,mutating=false,failurePolicy=fail,sideEffects=None,groups="",resources=secrets,verbs=update;delete,versions=v1,name=vsecret.kb.io,admissionReviewVersions={v1}

type secretValidator struct {
	client.Client
	dec *admission.Decoder
}

var _ admission.Handler = &secretValidator{}

// Handler validates Secrets.
func (v *secretValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	switch req.Operation {
	case admissionv1.Delete:
		return v.handleDelete(req)
	case admissionv1.Update:
		return v.handleUpdate(req)
	default:
		return admission.Allowed("")
	}
}

func (v *secretValidator) handleDelete(req admission.Request) admission.Response {
	s := &corev1.Secret{}
	if err := v.dec.DecodeRaw(req.OldObject, s); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	owners := s.GetOwnerReferences()
	for _, owner := range owners {
		if owner.APIVersion == dkimmanagerv1.GroupVersion.String() && owner.Kind == "DKIMKey" {
			for _, g := range req.UserInfo.Groups {
				if g == "system:serviceaccounts" {
					return admission.Allowed("")
				}
			}
			return admission.Denied("directly deleting DKIM private keys is not allowed")
		}
	}
	return admission.Allowed("")
}

func (v *secretValidator) handleUpdate(req admission.Request) admission.Response {
	s := &corev1.Secret{}
	if err := v.dec.Decode(req, s); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	owners := s.GetOwnerReferences()
	for _, owner := range owners {
		if owner.APIVersion == dkimmanagerv1.GroupVersion.String() && owner.Kind == "DKIMKey" {
			return admission.Denied("directly updating DKIM private keys is not allowed")
		}
	}
	return admission.Allowed("")
}

func SetupSecretWebhook(mgr manager.Manager, dec *admission.Decoder) {
	v := &secretValidator{
		Client: mgr.GetClient(),
		dec:    dec,
	}
	srv := mgr.GetWebhookServer()
	srv.Register("/validate-secret", &webhook.Admission{Handler: v})
}
