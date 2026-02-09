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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dkimmanagerv2 "github.com/hsn723/dkim-manager/api/v2"
	"github.com/hsn723/dkim-manager/pkg/dkim"
)

func TestConvertTo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		status             DKIMKeyStatus
		generation         int64
		wantConditionCount int
		wantCondStatus     metav1.ConditionStatus
		wantCondReason     string
		wantObservedGen    int64
	}{
		{
			name:               "OK",
			status:             DKIMKeyStatusOK,
			generation:         3,
			wantConditionCount: 1,
			wantCondStatus:     metav1.ConditionTrue,
			wantCondReason:     dkimmanagerv2.ReasonSucceeded,
			wantObservedGen:    3,
		},
		{
			name:               "Invalid",
			status:             DKIMKeyStatusInvalid,
			generation:         2,
			wantConditionCount: 1,
			wantCondStatus:     metav1.ConditionFalse,
			wantCondReason:     dkimmanagerv2.ReasonInvalid,
			wantObservedGen:    2,
		},
		{
			name:               "Empty",
			status:             "",
			wantConditionCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := &DKIMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-key",
					Namespace:  "default",
					Generation: tt.generation,
				},
				Spec: DKIMKeySpec{
					SecretName: "my-secret",
					Selector:   "selector1",
					Domain:     "example.com",
					TTL:        3600,
					KeyLength:  dkim.KeyLength2048,
					KeyType:    dkim.KeyTypeRSA,
				},
				Status: tt.status,
			}

			dst := &dkimmanagerv2.DKIMKey{}
			err := src.ConvertTo(dst)
			require.NoError(t, err)

			assert.Equal(t, src.Name, dst.Name)
			assert.Equal(t, src.Namespace, dst.Namespace)
			assert.Equal(t, src.Spec.SecretName, dst.Spec.SecretName)
			assert.Equal(t, src.Spec.Selector, dst.Spec.Selector)
			assert.Equal(t, src.Spec.Domain, dst.Spec.Domain)
			assert.Equal(t, src.Spec.TTL, dst.Spec.TTL)
			assert.Equal(t, src.Spec.KeyLength, dst.Spec.KeyLength)
			assert.Equal(t, src.Spec.KeyType, dst.Spec.KeyType)

			require.Len(t, dst.Status.Conditions, tt.wantConditionCount)
			if tt.wantConditionCount > 0 {
				assert.Equal(t, dkimmanagerv2.ConditionReady, dst.Status.Conditions[0].Type)
				assert.Equal(t, tt.wantCondStatus, dst.Status.Conditions[0].Status)
				assert.Equal(t, tt.wantCondReason, dst.Status.Conditions[0].Reason)
				assert.Equal(t, tt.wantObservedGen, dst.Status.ObservedGeneration)
				assert.Equal(t, tt.wantObservedGen, dst.Status.Conditions[0].ObservedGeneration)
			}
		})
	}
}

func TestConvertFrom(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		conditions []metav1.Condition
		observedGen int64
		wantStatus DKIMKeyStatus
	}{
		{
			name: "Ready",
			conditions: []metav1.Condition{
				{
					Type:   dkimmanagerv2.ConditionReady,
					Status: metav1.ConditionTrue,
					Reason: dkimmanagerv2.ReasonSucceeded,
				},
			},
			observedGen: 1,
			wantStatus:  DKIMKeyStatusOK,
		},
		{
			name: "NotReady",
			conditions: []metav1.Condition{
				{
					Type:   dkimmanagerv2.ConditionReady,
					Status: metav1.ConditionFalse,
					Reason: dkimmanagerv2.ReasonInvalid,
				},
			},
			wantStatus: DKIMKeyStatusInvalid,
		},
		{
			name:       "NoConditions",
			conditions: nil,
			wantStatus: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := &dkimmanagerv2.DKIMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-key",
					Namespace: "default",
				},
				Spec: dkimmanagerv2.DKIMKeySpec{
					SecretName: "my-secret",
					Selector:   "selector1",
					Domain:     "example.com",
					TTL:        3600,
					KeyLength:  dkim.KeyLength2048,
					KeyType:    dkim.KeyTypeRSA,
				},
				Status: dkimmanagerv2.DKIMKeyStatus{
					ObservedGeneration: tt.observedGen,
					Conditions:         tt.conditions,
				},
			}

			dst := &DKIMKey{}
			err := dst.ConvertFrom(src)
			require.NoError(t, err)

			assert.Equal(t, src.Name, dst.Name)
			assert.Equal(t, src.Namespace, dst.Namespace)
			assert.Equal(t, src.Spec.SecretName, dst.Spec.SecretName)
			assert.Equal(t, tt.wantStatus, dst.Status)
		})
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status DKIMKeyStatus
	}{
		{
			name:   "OK",
			status: DKIMKeyStatusOK,
		},
		{
			name:   "Invalid",
			status: DKIMKeyStatusInvalid,
		},
		{
			name:   "Empty",
			status: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			original := &DKIMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-key",
					Namespace: "default",
				},
				Spec: DKIMKeySpec{
					SecretName: "my-secret",
					Selector:   "selector1",
					Domain:     "example.com",
					TTL:        3600,
					KeyLength:  dkim.KeyLength2048,
					KeyType:    dkim.KeyTypeRSA,
				},
				Status: tt.status,
			}

			hub := &dkimmanagerv2.DKIMKey{}
			err := original.ConvertTo(hub)
			require.NoError(t, err)

			roundTripped := &DKIMKey{}
			err = roundTripped.ConvertFrom(hub)
			require.NoError(t, err)

			assert.Equal(t, original.Name, roundTripped.Name)
			assert.Equal(t, original.Namespace, roundTripped.Namespace)
			assert.Equal(t, original.Spec, roundTripped.Spec)
			assert.Equal(t, original.Status, roundTripped.Status)
		})
	}
}
