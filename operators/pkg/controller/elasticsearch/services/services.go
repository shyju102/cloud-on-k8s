// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package services

import (
	"strconv"

	"github.com/elastic/cloud-on-k8s/operators/pkg/apis/elasticsearch/v1alpha1"
	"github.com/elastic/cloud-on-k8s/operators/pkg/controller/common"
	"github.com/elastic/cloud-on-k8s/operators/pkg/controller/elasticsearch/label"
	"github.com/elastic/cloud-on-k8s/operators/pkg/controller/elasticsearch/name"
	"github.com/elastic/cloud-on-k8s/operators/pkg/controller/elasticsearch/network"
	"github.com/elastic/cloud-on-k8s/operators/pkg/utils/k8s"
	"github.com/elastic/cloud-on-k8s/operators/pkg/utils/stringsutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	globalServiceSuffix = ".svc.cluster.local"
)

// ExternalServiceName returns the name for the external service
// associated to this cluster
func ExternalServiceName(esName string) string {
	return name.HTTPService(esName)
}

// ExternalServiceURL returns the URL used to reach Elasticsearch's external endpoint
func ExternalServiceURL(es v1alpha1.Elasticsearch) string {
	return stringsutil.Concat("https://", ExternalServiceName(es.Name), ".", es.Namespace, globalServiceSuffix, ":", strconv.Itoa(network.HTTPPort))
}

// NewExternalService returns the external service associated to the given cluster
// It is used by users to perform requests against one of the cluster nodes.
func NewExternalService(es v1alpha1.Elasticsearch) *corev1.Service {
	nsn := k8s.ExtractNamespacedName(&es)
	var svc = corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   es.Namespace,
			Name:        ExternalServiceName(es.Name),
			Labels:      label.NewLabels(nsn),
			Annotations: es.Spec.HTTP.Service.Metadata.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Selector: label.NewLabels(nsn),
			Ports: []corev1.ServicePort{
				{
					Name:     "https",
					Protocol: corev1.ProtocolTCP,
					Port:     network.HTTPPort,
				},
			},
			SessionAffinity: corev1.ServiceAffinityNone,
			Type:            common.GetServiceType(es.Spec.HTTP.Service.Spec.Type),
		},
	}
	if svc.Spec.Type != corev1.ServiceTypeClusterIP {
		svc.Spec.ExternalTrafficPolicy = corev1.ServiceExternalTrafficPolicyTypeCluster
	}
	return &svc
}

// IsServiceReady checks if a service has one or more ready endpoints.
func IsServiceReady(c k8s.Client, service corev1.Service) (bool, error) {
	endpoints := corev1.Endpoints{}
	namespacedName := types.NamespacedName{Namespace: service.Namespace, Name: service.Name}

	if err := c.Get(namespacedName, &endpoints); err != nil {
		return false, err
	}
	for _, subs := range endpoints.Subsets {
		if len(subs.Addresses) > 0 {
			return true, nil
		}
	}
	return false, nil
}

// GetExternalService returns the external service associated to the given Elasticsearch cluster.
func GetExternalService(c k8s.Client, es v1alpha1.Elasticsearch) (corev1.Service, error) {
	var svc corev1.Service

	namespacedName := types.NamespacedName{
		Namespace: es.Namespace,
		Name:      ExternalServiceName(es.Name),
	}

	if err := c.Get(namespacedName, &svc); err != nil {
		return corev1.Service{}, err
	}

	return svc, nil
}
