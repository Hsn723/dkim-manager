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

func TestConvertTo_OK(t *testing.T) {
	src := &DKIMKey{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-key",
			Namespace:  "default",
			Generation: 3,
		},
		Spec: DKIMKeySpec{
			SecretName: "my-secret",
			Selector:   "selector1",
			Domain:     "example.com",
			TTL:        3600,
			KeyLength:  dkim.KeyLength2048,
			KeyType:    dkim.KeyTypeRSA,
		},
		Status: DKIMKeyStatusOK,
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

	require.Len(t, dst.Status.Conditions, 1)
	assert.Equal(t, dkimmanagerv2.ConditionReady, dst.Status.Conditions[0].Type)
	assert.Equal(t, metav1.ConditionTrue, dst.Status.Conditions[0].Status)
	assert.Equal(t, dkimmanagerv2.ReasonSucceeded, dst.Status.Conditions[0].Reason)
	assert.Equal(t, int64(3), dst.Status.ObservedGeneration)
	assert.Equal(t, int64(3), dst.Status.Conditions[0].ObservedGeneration)
}

func TestConvertTo_Invalid(t *testing.T) {
	src := &DKIMKey{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-key",
			Namespace:  "default",
			Generation: 2,
		},
		Spec: DKIMKeySpec{
			SecretName: "my-secret",
			Selector:   "selector1",
			Domain:     "example.com",
		},
		Status: DKIMKeyStatusInvalid,
	}

	dst := &dkimmanagerv2.DKIMKey{}
	err := src.ConvertTo(dst)
	require.NoError(t, err)

	require.Len(t, dst.Status.Conditions, 1)
	assert.Equal(t, dkimmanagerv2.ConditionReady, dst.Status.Conditions[0].Type)
	assert.Equal(t, metav1.ConditionFalse, dst.Status.Conditions[0].Status)
	assert.Equal(t, dkimmanagerv2.ReasonInvalid, dst.Status.Conditions[0].Reason)
	assert.Equal(t, int64(2), dst.Status.ObservedGeneration)
	assert.Equal(t, int64(2), dst.Status.Conditions[0].ObservedGeneration)
}

func TestConvertTo_Empty(t *testing.T) {
	src := &DKIMKey{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-key",
			Namespace: "default",
		},
		Spec: DKIMKeySpec{
			SecretName: "my-secret",
			Selector:   "selector1",
			Domain:     "example.com",
		},
		Status: "",
	}

	dst := &dkimmanagerv2.DKIMKey{}
	err := src.ConvertTo(dst)
	require.NoError(t, err)

	assert.Empty(t, dst.Status.Conditions)
}

func TestConvertFrom_Ready(t *testing.T) {
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
			ObservedGeneration: 1,
			Conditions: []metav1.Condition{
				{
					Type:   dkimmanagerv2.ConditionReady,
					Status: metav1.ConditionTrue,
					Reason: dkimmanagerv2.ReasonSucceeded,
				},
			},
		},
	}

	dst := &DKIMKey{}
	err := dst.ConvertFrom(src)
	require.NoError(t, err)

	assert.Equal(t, src.Name, dst.Name)
	assert.Equal(t, src.Namespace, dst.Namespace)
	assert.Equal(t, src.Spec.SecretName, dst.Spec.SecretName)
	assert.Equal(t, DKIMKeyStatusOK, dst.Status)
}

func TestConvertFrom_NotReady(t *testing.T) {
	src := &dkimmanagerv2.DKIMKey{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-key",
			Namespace: "default",
		},
		Spec: dkimmanagerv2.DKIMKeySpec{
			SecretName: "my-secret",
			Selector:   "selector1",
			Domain:     "example.com",
		},
		Status: dkimmanagerv2.DKIMKeyStatus{
			Conditions: []metav1.Condition{
				{
					Type:   dkimmanagerv2.ConditionReady,
					Status: metav1.ConditionFalse,
					Reason: dkimmanagerv2.ReasonInvalid,
				},
			},
		},
	}

	dst := &DKIMKey{}
	err := dst.ConvertFrom(src)
	require.NoError(t, err)

	assert.Equal(t, DKIMKeyStatusInvalid, dst.Status)
}

func TestConvertFrom_NoConditions(t *testing.T) {
	src := &dkimmanagerv2.DKIMKey{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-key",
			Namespace: "default",
		},
		Spec: dkimmanagerv2.DKIMKeySpec{
			SecretName: "my-secret",
			Selector:   "selector1",
			Domain:     "example.com",
		},
		Status: dkimmanagerv2.DKIMKeyStatus{},
	}

	dst := &DKIMKey{}
	err := dst.ConvertFrom(src)
	require.NoError(t, err)

	assert.Equal(t, DKIMKeyStatus(""), dst.Status)
}

func TestRoundTrip_OK(t *testing.T) {
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
		Status: DKIMKeyStatusOK,
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
}

func TestRoundTrip_Invalid(t *testing.T) {
	original := &DKIMKey{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-key",
			Namespace: "default",
		},
		Spec: DKIMKeySpec{
			SecretName: "my-secret",
			Selector:   "selector1",
			Domain:     "example.com",
		},
		Status: DKIMKeyStatusInvalid,
	}

	hub := &dkimmanagerv2.DKIMKey{}
	err := original.ConvertTo(hub)
	require.NoError(t, err)

	roundTripped := &DKIMKey{}
	err = roundTripped.ConvertFrom(hub)
	require.NoError(t, err)

	assert.Equal(t, original.Status, roundTripped.Status)
}

func TestRoundTrip_Empty(t *testing.T) {
	original := &DKIMKey{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-key",
			Namespace: "default",
		},
		Spec: DKIMKeySpec{
			SecretName: "my-secret",
			Selector:   "selector1",
			Domain:     "example.com",
		},
		Status: "",
	}

	hub := &dkimmanagerv2.DKIMKey{}
	err := original.ConvertTo(hub)
	require.NoError(t, err)

	roundTripped := &DKIMKey{}
	err = roundTripped.ConvertFrom(hub)
	require.NoError(t, err)

	assert.Equal(t, original.Status, roundTripped.Status)
}
