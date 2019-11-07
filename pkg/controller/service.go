/*
Copyright 2018 The Knative Authors

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

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/serving/pkg/apis/serving"
	v1alpha12 "knative.dev/serving/pkg/apis/serving/v1"
	versioned "knative.dev/serving/pkg/client/clientset/versioned"
	listers "knative.dev/serving/pkg/client/listers/serving/v1"
	"knative.dev/serving/pkg/reconciler"
	resourcenames "knative.dev/serving/pkg/reconciler/service/resources/names"
)

const (
	// ReconcilerName is the name of the reconciler
	ReconcilerName = "serving-controller"
)

// Reconciler implements controller.Reconciler for Service resources.
type Reconciler struct {
	*reconciler.Base

	// listers index properties about resources
	serviceLister     listers.ServiceLister
	revisionLister    listers.RevisionLister
	routeLister       listers.RouteLister
	revisionClientSet versioned.Interface
}

// Check that our Reconciler implements controller.Reconciler
var _ controller.Reconciler = (*Reconciler)(nil)

// Reconcile compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Service resource
// with the current status of the resource.
func (c *Reconciler) Reconcile(ctx context.Context, key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		c.Logger.Errorf("invalid resource key: %s", key)
		return nil
	}
	logger := logging.FromContext(ctx)

	logger.Infof("Reconcile: %s/%s", namespace, name)

	// Get the Service resource with this namespace/name
	original, err := c.serviceLister.Services(namespace).Get(name)
	if apierrs.IsNotFound(err) {
		// The resource may no longer exist, in which case we stop processing.
		logger.Errorf("service %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		return err
	}

	if original.GetDeletionTimestamp() != nil {
		return nil
	}

	// Don't modify the informers copy
	service := original.DeepCopy()

	// Reconcile this copy of the service and then write back any status
	// updates regardless of whether the reconciliation errored out.
	if reconcileErr := c.reconcile(ctx, service); reconcileErr != nil {
		c.Recorder.Event(service, corev1.EventTypeWarning, "InternalError", reconcileErr.Error())
		logger.Errorf("Reconcile service: %s/%s error: %s ", service.Namespace, service.Name, reconcileErr.Error())
		return reconcileErr
	}

	return nil
}

func (c *Reconciler) reconcile(ctx context.Context, service *v1alpha12.Service) error {
	logger := logging.FromContext(ctx)

	routeName := resourcenames.Route(service)
	route, err := c.routeLister.Routes(service.Namespace).Get(routeName)
	if apierrs.IsNotFound(err) {
		logger.Infof("controller reconcile service: %s/%s route is not found", service.Namespace, service.Name)
		return nil
	}
	logger.Infof("service: %s/%s route: %s/%s ", service.Namespace, service.Name, route.Namespace, route.Name)

	revisions, err := c.revisionLister.Revisions(service.Namespace).List(labels.SelectorFromSet(map[string]string{
		serving.ServiceLabelKey:       service.Name,
		serving.ConfigurationLabelKey: resourcenames.Configuration(service),
	}))
	if err != nil {
		logger.Infof("controller reconcile service: %s/%s get revisions error:%s", service.Namespace, service.Name, err.Error())
		return err
	}
	logger.Infof("service: %s/%s revisions: %d ", service.Namespace, service.Name, len(revisions))
	for _, re := range revisions {
		logger.Infof("service: %s/%s revision: %s/%s ", service.Namespace, service.Name, re.Namespace, re.Name)
	}

	return nil
}
