/*
Copyright AppsCode Inc. and Contributors

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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	autoscalingv1alpha1 "kubedb.dev/apimachinery/apis/autoscaling/v1alpha1"
	versioned "kubedb.dev/apimachinery/client/clientset/versioned"
	internalinterfaces "kubedb.dev/apimachinery/client/informers/externalversions/internalinterfaces"
	v1alpha1 "kubedb.dev/apimachinery/client/listers/autoscaling/v1alpha1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// MariaDBAutoscalerInformer provides access to a shared informer and lister for
// MariaDBAutoscalers.
type MariaDBAutoscalerInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.MariaDBAutoscalerLister
}

type mariaDBAutoscalerInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewMariaDBAutoscalerInformer constructs a new informer for MariaDBAutoscaler type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewMariaDBAutoscalerInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredMariaDBAutoscalerInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredMariaDBAutoscalerInformer constructs a new informer for MariaDBAutoscaler type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredMariaDBAutoscalerInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AutoscalingV1alpha1().MariaDBAutoscalers(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AutoscalingV1alpha1().MariaDBAutoscalers(namespace).Watch(context.TODO(), options)
			},
		},
		&autoscalingv1alpha1.MariaDBAutoscaler{},
		resyncPeriod,
		indexers,
	)
}

func (f *mariaDBAutoscalerInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredMariaDBAutoscalerInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *mariaDBAutoscalerInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&autoscalingv1alpha1.MariaDBAutoscaler{}, f.defaultInformer)
}

func (f *mariaDBAutoscalerInformer) Lister() v1alpha1.MariaDBAutoscalerLister {
	return v1alpha1.NewMariaDBAutoscalerLister(f.Informer().GetIndexer())
}
