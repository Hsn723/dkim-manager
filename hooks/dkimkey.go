package hooks

import (
	"context"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	dkimmanagerv1 "github.com/hsn723/dkim-manager/api/v1"
)

//+kubebuilder:webhook:path=/validate-dkim-manager-atelierhsn-com-v1-dkimkey,mutating=false,failurePolicy=fail,sideEffects=None,groups=dkim-manager.atelierhsn.com,resources=dkimkeys,verbs=update,versions=v1,name=vdkimkey.kb.io,admissionReviewVersions={v1}

type dkimKeyValidator struct {
	client.Client
	dec *admission.Decoder
}

var _ admission.Handler = &dkimKeyValidator{}

// Handler validates DKIMKeys.
func (v *dkimKeyValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	switch req.Operation {
	case admissionv1.Update:
		return v.handleUpdate(req)
	default:
		return admission.Allowed("")
	}
}

func (v *dkimKeyValidator) handleUpdate(req admission.Request) admission.Response {
	dkNew := &dkimmanagerv1.DKIMKey{}
	decoder := *v.dec
	if err := decoder.Decode(req, dkNew); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	dkOld := &dkimmanagerv1.DKIMKey{}
	if err := decoder.DecodeRaw(req.OldObject, dkOld); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if dkNew.Name != dkOld.Name {
		return admission.Denied("changing dkimkey name is not allowed")
	}
	if !equality.Semantic.DeepEqual(dkNew.Spec, dkOld.Spec) {
		return admission.Denied("changing dkimkey spec is not allowed")
	}
	return admission.Allowed("")
}

func SetupDKIMKeyWebhook(mgr manager.Manager, dec *admission.Decoder) {
	v := &dkimKeyValidator{
		Client: mgr.GetClient(),
		dec:    dec,
	}
	srv := mgr.GetWebhookServer()
	srv.Register("/validate-dkim-manager-atelierhsn-com-v1-dkimkey", &webhook.Admission{Handler: v})
}
