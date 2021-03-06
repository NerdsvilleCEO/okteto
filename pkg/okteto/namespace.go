// Copyright 2020 The Okteto Authors
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

package okteto

import (
	"context"
	"fmt"
)

// CreateBody top body answer
type CreateBody struct {
	Namespace Namespace `json:"createSpace" yaml:"createSpace"`
}

// DeleteBody top body answer
type DeleteBody struct {
	Namespace Namespace `json:"deleteSpace" yaml:"deleteSpace"`
}

//Namespace represents an Okteto k8s namespace
type Namespace struct {
	ID string `json:"id" yaml:"id"`
}

// CreateNamespace creates a namespace
func CreateNamespace(ctx context.Context, namespace string) (string, error) {
	q := fmt.Sprintf(`mutation{
		createSpace(name: "%s"){
			id
		},
	}`, namespace)

	var body CreateBody
	if err := query(ctx, q, &body); err != nil {
		return "", err
	}

	return body.Namespace.ID, nil
}

// DeleteNamespace deletes a namespace
func DeleteNamespace(ctx context.Context, namespace string) error {
	q := fmt.Sprintf(`mutation{
		deleteSpace(id: "%s"){
			id
		},
	}`, namespace)

	var body DeleteBody
	if err := query(ctx, q, &body); err != nil {
		return err
	}

	return nil
}
