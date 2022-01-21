package externaldns

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func DNSEndpoint() *unstructured.Unstructured {
	de := &unstructured.Unstructured{}
	de.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   DNSEndpointGroup,
		Version: DNSEndpointVersion,
		Kind:    "DNSEndpoint",
	})
	return de
}

func DNSEndpointList() *unstructured.UnstructuredList {
	del := &unstructured.UnstructuredList{}
	del.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   DNSEndpointGroup,
		Version: DNSEndpointVersion,
		Kind:    "DNSEndpointList",
	})
	return del
}
