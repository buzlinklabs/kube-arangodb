//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
package fake

import (
	v1alpha "github.com/arangodb/k8s-operator/pkg/apis/arangodb/v1alpha"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeArangoDeployments implements ArangoDeploymentInterface
type FakeArangoDeployments struct {
	Fake *FakeDatabaseV1alpha
	ns   string
}

var arangodeploymentsResource = schema.GroupVersionResource{Group: "database.arangodb.com", Version: "v1alpha", Resource: "arangodeployments"}

var arangodeploymentsKind = schema.GroupVersionKind{Group: "database.arangodb.com", Version: "v1alpha", Kind: "ArangoDeployment"}

// Get takes name of the arangoDeployment, and returns the corresponding arangoDeployment object, and an error if there is any.
func (c *FakeArangoDeployments) Get(name string, options v1.GetOptions) (result *v1alpha.ArangoDeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(arangodeploymentsResource, c.ns, name), &v1alpha.ArangoDeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha.ArangoDeployment), err
}

// List takes label and field selectors, and returns the list of ArangoDeployments that match those selectors.
func (c *FakeArangoDeployments) List(opts v1.ListOptions) (result *v1alpha.ArangoDeploymentList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(arangodeploymentsResource, arangodeploymentsKind, c.ns, opts), &v1alpha.ArangoDeploymentList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha.ArangoDeploymentList{}
	for _, item := range obj.(*v1alpha.ArangoDeploymentList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested arangoDeployments.
func (c *FakeArangoDeployments) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(arangodeploymentsResource, c.ns, opts))

}

// Create takes the representation of a arangoDeployment and creates it.  Returns the server's representation of the arangoDeployment, and an error, if there is any.
func (c *FakeArangoDeployments) Create(arangoDeployment *v1alpha.ArangoDeployment) (result *v1alpha.ArangoDeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(arangodeploymentsResource, c.ns, arangoDeployment), &v1alpha.ArangoDeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha.ArangoDeployment), err
}

// Update takes the representation of a arangoDeployment and updates it. Returns the server's representation of the arangoDeployment, and an error, if there is any.
func (c *FakeArangoDeployments) Update(arangoDeployment *v1alpha.ArangoDeployment) (result *v1alpha.ArangoDeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(arangodeploymentsResource, c.ns, arangoDeployment), &v1alpha.ArangoDeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha.ArangoDeployment), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeArangoDeployments) UpdateStatus(arangoDeployment *v1alpha.ArangoDeployment) (*v1alpha.ArangoDeployment, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(arangodeploymentsResource, "status", c.ns, arangoDeployment), &v1alpha.ArangoDeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha.ArangoDeployment), err
}

// Delete takes name of the arangoDeployment and deletes it. Returns an error if one occurs.
func (c *FakeArangoDeployments) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(arangodeploymentsResource, c.ns, name), &v1alpha.ArangoDeployment{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeArangoDeployments) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(arangodeploymentsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha.ArangoDeploymentList{})
	return err
}

// Patch applies the patch and returns the patched arangoDeployment.
func (c *FakeArangoDeployments) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha.ArangoDeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(arangodeploymentsResource, c.ns, name, data, subresources...), &v1alpha.ArangoDeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha.ArangoDeployment), err
}
