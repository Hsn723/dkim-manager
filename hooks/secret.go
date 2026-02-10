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
)

//+kubebuilder:webhook:path=/validate-secret,mutating=false,failurePolicy=fail,sideEffects=None,groups="",resources=secrets,verbs=delete,versions=v1,name=vsecret.kb.io,admissionReviewVersions={v1}

type secretValidator struct {
	client.Client
	dec                *admission.Decoder
	serviceAccountName string
}

var _ admission.Handler = &secretValidator{}

// Handler validates Secrets.
func (v *secretValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	switch req.Operation {
	case admissionv1.Delete:
		return v.handleDelete(req)
	default:
		return admission.Allowed("")
	}
}

func (v *secretValidator) handleDelete(req admission.Request) admission.Response {
	s := &corev1.Secret{}
	if err := (*v.dec).DecodeRaw(req.OldObject, s); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	owners := s.GetOwnerReferences()
	for _, owner := range owners {
		if isDKIMKeyOwner(owner) {
			if req.UserInfo.Username == v.serviceAccountName {
				return admission.Allowed("deletion by service account allowed")
			}
			return admission.Denied("directly deleting DKIM private keys is not allowed")
		}
	}
	return admission.Allowed("")
}

func SetupSecretWebhook(mgr manager.Manager, dec *admission.Decoder, sa string) {
	v := &secretValidator{
		Client:             mgr.GetClient(),
		dec:                dec,
		serviceAccountName: sa,
	}
	srv := mgr.GetWebhookServer()
	srv.Register("/validate-secret", &webhook.Admission{Handler: v})
}
