package hooks

import (
	"context"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	dkimmanagerv1 "github.com/hsn723/dkim-manager/api/v1"
	"github.com/hsn723/dkim-manager/pkg/externaldns"
)

//+kubebuilder:webhook:path=/validate-externaldns-k8s-io-v1alpha1-dnsendpoint,mutating=false,failurePolicy=fail,sideEffects=None,groups=externaldns.k8s.io,resources=dnsendpoints,verbs=delete,versions=v1alpha1,name=vdnsendpoint.kb.io,admissionReviewVersions={v1}

type dnsEndpointValidator struct {
	client.Client
	dec                *admission.Decoder
	serviceAccountName string
}

var _ admission.Handler = &dnsEndpointValidator{}

// Handler validates DNSEndpoints.
func (v *dnsEndpointValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	switch req.Operation {
	case admissionv1.Delete:
		return v.handleDelete(req)
	default:
		return admission.Allowed("")
	}
}

func (v *dnsEndpointValidator) handleDelete(req admission.Request) admission.Response {
	de := externaldns.DNSEndpoint()
	if err := v.dec.DecodeRaw(req.OldObject, de); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	owners := de.GetOwnerReferences()
	for _, owner := range owners {
		if owner.APIVersion == dkimmanagerv1.GroupVersion.String() && owner.Kind == dkimmanagerv1.DKIMKeyKind {
			if req.UserInfo.Username == v.serviceAccountName {
				return admission.Allowed("deletion by service account allowed")
			}
			return admission.Denied("directly deleting DKIM record is not allowed")
		}
	}
	return admission.Allowed("")
}

func SetupDNSEndpointWebhook(mgr manager.Manager, dec *admission.Decoder, sa string) {
	v := &dnsEndpointValidator{
		Client:             mgr.GetClient(),
		dec:                dec,
		serviceAccountName: sa,
	}
	srv := mgr.GetWebhookServer()
	srv.Register("/validate-externaldns-k8s-io-v1alpha1-dnsendpoint", &webhook.Admission{Handler: v})
}
