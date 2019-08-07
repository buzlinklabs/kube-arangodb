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

package backup

import (
	"time"

	database "github.com/arangodb/kube-arangodb/pkg/apis/deployment/v1alpha"
	"github.com/arangodb/kube-arangodb/pkg/backup/operator"
	arangoClientSet "github.com/arangodb/kube-arangodb/pkg/generated/clientset/versioned"
	arangoInformer "github.com/arangodb/kube-arangodb/pkg/generated/informers/externalversions"
)

func registerInformer(operator operator.Operator, informer arangoInformer.SharedInformerFactory) error {
	if err := operator.RegisterInformer(informer.Database().V1alpha().ArangoBackups().Informer(),
		database.SchemeGroupVersion.Group,
		database.SchemeGroupVersion.Version,
		database.ArangoBackupResourceKind); err != nil {
		return err
	}

	return nil
}

func RegisterHandler(operator operator.Operator, client arangoClientSet.Interface) error {
	informer := arangoInformer.NewSharedInformerFactory(client, 30*time.Second)

	if err := registerInformer(operator, informer); err != nil {
		return err
	}

	if err := operator.RegisterStarter(informer); err != nil {
		return err
	}

	h := &handler{
		client: client,
	}

	if err := operator.RegisterHandler(h); err != nil {
		return err
	}

	return nil
}
