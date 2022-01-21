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
	"github.com/hsn723/dkim-manager/pkg/dkim"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DKIMKeySpec defines the desired state of DKIMKey.
type DKIMKeySpec struct {
	// SecretName represents the name for the Secret resource containing the private key.
	SecretName string `json:"secretName"`

	// Selector is the name to use as a DKIM selector.
	Selector string `json:"selector"`

	// Domain is the domain to which the DKIM record will be associated.
	Domain string `json:"domain"`

	// +kubebuilder:default=86400

	// TTL for the DKIM record.
	TTL uint `json:"ttl,omitempty"`

	// +kubebuilder:validation:Enum=1024;2048;4096
	// +kubebuilder:default=2048

	// KeyLength represents the bit size for RSA keys.
	KeyLength dkim.KeyLength `json:"keyLength,omitempty"`

	// +kubebuilder:validation:Enum=rsa;ed25519
	// +kubebuilder:default=rsa

	// KeyType represents the DKIM key type.
	KeyType dkim.KeyType `json:"keyType,omitempty"`
}

// DKIMKeyStatus defines the observed state of DKIMKey.
type DKIMKeyStatus string

const (
	DKIMKeyStatusOK = DKIMKeyStatus("ok")
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DKIMKey is the Schema for the dkimkeys API.
type DKIMKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DKIMKeySpec   `json:"spec"`
	Status DKIMKeyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DKIMKeyList contains a list of DKIMKey.
type DKIMKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DKIMKey `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DKIMKey{}, &DKIMKeyList{})
}
