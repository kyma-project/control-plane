/*
Copyright (c) SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/gardener/pkg/apis/core/v1alpha1"
	scheme "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/gardener/pkg/client/core/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ShootExtensionStatusesGetter has a method to return a ShootExtensionStatusInterface.
// A group's client should implement this interface.
type ShootExtensionStatusesGetter interface {
	ShootExtensionStatuses(namespace string) ShootExtensionStatusInterface
}

// ShootExtensionStatusInterface has methods to work with ShootExtensionStatus resources.
type ShootExtensionStatusInterface interface {
	Create(ctx context.Context, shootExtensionStatus *v1alpha1.ShootExtensionStatus, opts v1.CreateOptions) (*v1alpha1.ShootExtensionStatus, error)
	Update(ctx context.Context, shootExtensionStatus *v1alpha1.ShootExtensionStatus, opts v1.UpdateOptions) (*v1alpha1.ShootExtensionStatus, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.ShootExtensionStatus, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.ShootExtensionStatusList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ShootExtensionStatus, err error)
	ShootExtensionStatusExpansion
}

// shootExtensionStatuses implements ShootExtensionStatusInterface
type shootExtensionStatuses struct {
	client rest.Interface
	ns     string
}

// newShootExtensionStatuses returns a ShootExtensionStatuses
func newShootExtensionStatuses(c *CoreV1alpha1Client, namespace string) *shootExtensionStatuses {
	return &shootExtensionStatuses{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the shootExtensionStatus, and returns the corresponding shootExtensionStatus object, and an error if there is any.
func (c *shootExtensionStatuses) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.ShootExtensionStatus, err error) {
	result = &v1alpha1.ShootExtensionStatus{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("shootextensionstatuses").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ShootExtensionStatuses that match those selectors.
func (c *shootExtensionStatuses) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ShootExtensionStatusList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.ShootExtensionStatusList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("shootextensionstatuses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested shootExtensionStatuses.
func (c *shootExtensionStatuses) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("shootextensionstatuses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a shootExtensionStatus and creates it.  Returns the server's representation of the shootExtensionStatus, and an error, if there is any.
func (c *shootExtensionStatuses) Create(ctx context.Context, shootExtensionStatus *v1alpha1.ShootExtensionStatus, opts v1.CreateOptions) (result *v1alpha1.ShootExtensionStatus, err error) {
	result = &v1alpha1.ShootExtensionStatus{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("shootextensionstatuses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(shootExtensionStatus).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a shootExtensionStatus and updates it. Returns the server's representation of the shootExtensionStatus, and an error, if there is any.
func (c *shootExtensionStatuses) Update(ctx context.Context, shootExtensionStatus *v1alpha1.ShootExtensionStatus, opts v1.UpdateOptions) (result *v1alpha1.ShootExtensionStatus, err error) {
	result = &v1alpha1.ShootExtensionStatus{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("shootextensionstatuses").
		Name(shootExtensionStatus.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(shootExtensionStatus).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the shootExtensionStatus and deletes it. Returns an error if one occurs.
func (c *shootExtensionStatuses) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("shootextensionstatuses").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *shootExtensionStatuses) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("shootextensionstatuses").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched shootExtensionStatus.
func (c *shootExtensionStatuses) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ShootExtensionStatus, err error) {
	result = &v1alpha1.ShootExtensionStatus{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("shootextensionstatuses").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
