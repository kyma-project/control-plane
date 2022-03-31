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

// ExposureClassesGetter has a method to return a ExposureClassInterface.
// A group's client should implement this interface.
type ExposureClassesGetter interface {
	ExposureClasses() ExposureClassInterface
}

// ExposureClassInterface has methods to work with ExposureClass resources.
type ExposureClassInterface interface {
	Create(ctx context.Context, exposureClass *v1alpha1.ExposureClass, opts v1.CreateOptions) (*v1alpha1.ExposureClass, error)
	Update(ctx context.Context, exposureClass *v1alpha1.ExposureClass, opts v1.UpdateOptions) (*v1alpha1.ExposureClass, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.ExposureClass, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.ExposureClassList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ExposureClass, err error)
	ExposureClassExpansion
}

// exposureClasses implements ExposureClassInterface
type exposureClasses struct {
	client rest.Interface
}

// newExposureClasses returns a ExposureClasses
func newExposureClasses(c *CoreV1alpha1Client) *exposureClasses {
	return &exposureClasses{
		client: c.RESTClient(),
	}
}

// Get takes name of the exposureClass, and returns the corresponding exposureClass object, and an error if there is any.
func (c *exposureClasses) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.ExposureClass, err error) {
	result = &v1alpha1.ExposureClass{}
	err = c.client.Get().
		Resource("exposureclasses").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ExposureClasses that match those selectors.
func (c *exposureClasses) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ExposureClassList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.ExposureClassList{}
	err = c.client.Get().
		Resource("exposureclasses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested exposureClasses.
func (c *exposureClasses) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("exposureclasses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a exposureClass and creates it.  Returns the server's representation of the exposureClass, and an error, if there is any.
func (c *exposureClasses) Create(ctx context.Context, exposureClass *v1alpha1.ExposureClass, opts v1.CreateOptions) (result *v1alpha1.ExposureClass, err error) {
	result = &v1alpha1.ExposureClass{}
	err = c.client.Post().
		Resource("exposureclasses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(exposureClass).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a exposureClass and updates it. Returns the server's representation of the exposureClass, and an error, if there is any.
func (c *exposureClasses) Update(ctx context.Context, exposureClass *v1alpha1.ExposureClass, opts v1.UpdateOptions) (result *v1alpha1.ExposureClass, err error) {
	result = &v1alpha1.ExposureClass{}
	err = c.client.Put().
		Resource("exposureclasses").
		Name(exposureClass.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(exposureClass).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the exposureClass and deletes it. Returns an error if one occurs.
func (c *exposureClasses) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("exposureclasses").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *exposureClasses) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("exposureclasses").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched exposureClass.
func (c *exposureClasses) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ExposureClass, err error) {
	result = &v1alpha1.ExposureClass{}
	err = c.client.Patch(pt).
		Resource("exposureclasses").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
