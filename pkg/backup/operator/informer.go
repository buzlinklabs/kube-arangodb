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
// Author Adam Janikowski
//

package operator

import (
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

func newResourceEventHandler(operator Operator, group, version, kind string) cache.ResourceEventHandler {
	return &resourceEventWrapper{
		Operator: operator,
		Group:    group,
		Version:  version,
		Kind:     kind,
	}
}

type resourceEventWrapper struct {
	Operator Operator

	Group, Version, Kind string
}

func (r *resourceEventWrapper) push(operation Operation, obj interface{}) {
	if obj == nil {
		return
	}

	if object, ok := obj.(meta.Object); ok {
		if item, err := NewItemFromObject(operation, r.Group, r.Version, r.Kind, object); err == nil {
			r.Operator.EnqueueItem(item)
		}
	}
}

func (r *resourceEventWrapper) OnAdd(obj interface{}) {
	r.push(OperationAdd, obj)
}

func (r *resourceEventWrapper) OnUpdate(oldObj, newObj interface{}) {
	r.push(OperationUpdate, newObj)
}

func (r *resourceEventWrapper) OnDelete(obj interface{}) {
	r.push(OperationDelete, obj)
}