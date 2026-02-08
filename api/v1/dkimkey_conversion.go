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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	dkimmanagerv2 "github.com/hsn723/dkim-manager/api/v2"
)

// ConvertTo converts this DKIMKey (v1) to the Hub version (v2).
func (src *DKIMKey) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*dkimmanagerv2.DKIMKey)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec
	dst.Spec = dkimmanagerv2.DKIMKeySpec{
		SecretName: src.Spec.SecretName,
		Selector:   src.Spec.Selector,
		Domain:     src.Spec.Domain,
		TTL:        src.Spec.TTL,
		KeyLength:  src.Spec.KeyLength,
		KeyType:    src.Spec.KeyType,
	}

	// Status: convert string -> conditions
	switch src.Status {
	case DKIMKeyStatusOK:
		dst.Status = dkimmanagerv2.DKIMKeyStatus{
			ObservedGeneration: src.Generation,
			Conditions: []metav1.Condition{
				{
					Type:               dkimmanagerv2.ConditionReady,
					Status:             metav1.ConditionTrue,
					Reason:             dkimmanagerv2.ReasonSucceeded,
					Message:            "DKIM key created successfully",
					ObservedGeneration: src.Generation,
					LastTransitionTime: metav1.Now(),
				},
			},
		}
	case DKIMKeyStatusInvalid:
		dst.Status = dkimmanagerv2.DKIMKeyStatus{
			ObservedGeneration: src.Generation,
			Conditions: []metav1.Condition{
				{
					Type:               dkimmanagerv2.ConditionReady,
					Status:             metav1.ConditionFalse,
					Reason:             dkimmanagerv2.ReasonInvalid,
					Message:            "DKIMKey is invalid",
					ObservedGeneration: src.Generation,
					LastTransitionTime: metav1.Now(),
				},
			},
		}
	default:
		// empty status -> no conditions
		dst.Status = dkimmanagerv2.DKIMKeyStatus{}
	}

	return nil
}

// ConvertFrom converts from the Hub version (v2) to this version (v1).
func (dst *DKIMKey) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*dkimmanagerv2.DKIMKey)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec
	dst.Spec = DKIMKeySpec{
		SecretName: src.Spec.SecretName,
		Selector:   src.Spec.Selector,
		Domain:     src.Spec.Domain,
		TTL:        src.Spec.TTL,
		KeyLength:  src.Spec.KeyLength,
		KeyType:    src.Spec.KeyType,
	}

	// Status: convert conditions -> string
	if src.IsReady() {
		dst.Status = DKIMKeyStatusOK
	} else if len(src.Status.Conditions) > 0 {
		dst.Status = DKIMKeyStatusInvalid
	} else {
		dst.Status = ""
	}

	return nil
}
