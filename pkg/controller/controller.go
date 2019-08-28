/*
Copyright 2019 The Knative Authors

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

	servingclient "knative.dev/serving/pkg/client/injection/client"
	configurationinformer "knative.dev/serving/pkg/client/injection/informers/serving/v1alpha1/configuration"
	revisioninformer "knative.dev/serving/pkg/client/injection/informers/serving/v1alpha1/revision"
	routeinformer "knative.dev/serving/pkg/client/injection/informers/serving/v1alpha1/route"
	kserviceinformer "knative.dev/serving/pkg/client/injection/informers/serving/v1alpha1/service"

	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/serving/pkg/apis/serving/v1alpha1"
)

// NewController initializes the controller and is called by the generated code
// Registers eventhandlers to enqueue events
func NewController(
	ctx context.Context,
) *controller.Impl {
	logger := logging.FromContext(ctx)
	serviceInformer := kserviceinformer.Get(ctx)
	routeInformer := routeinformer.Get(ctx)
	configurationInformer := configurationinformer.Get(ctx)
	revisionInformer := revisioninformer.Get(ctx)

	c := &Reconciler{
		serviceLister:     serviceInformer.Lister(),
		revisionLister:    revisionInformer.Lister(),
		revisionClientSet: servingclient.Get(ctx),
		routeLister:       routeInformer.Lister(),
	}
	impl := controller.NewImpl(c, logger, ReconcilerName)

	logger.Info("Setting up event handlers")
	serviceInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	configurationInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.Filter(v1alpha1.SchemeGroupVersion.WithKind("Service")),
		Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	})

	routeInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.Filter(v1alpha1.SchemeGroupVersion.WithKind("Service")),
		Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	})

	return impl
}
