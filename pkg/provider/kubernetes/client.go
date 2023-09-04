// Copyright © 2023 Cisco
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kubernetes

import (
	"context"
	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientv1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type client struct {
	namespace     string
	secretsClient clientv1.SecretInterface
}

func (c *client) GetSecret(ctx context.Context, key v1alpha1.SecretKey) ([]byte, error) {
	if len(key.GetPath()) > 0 {
		return nil, v1alpha1.ErrKeyPathUnsupported
	}
	secret, err := c.secretsClient.Get(ctx, key.Key, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret.Data[key.Property], nil
}

func (c *client) ListSecretKeys(ctx context.Context, query v1alpha1.SecretKeyQuery) ([]v1alpha1.SecretKey, error) {
	listData, err := c.secretsClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var keys []v1alpha1.SecretKey
	for _, secret := range listData.Items {
		for property := range secret.Data {
			keys = append(keys, v1alpha1.SecretKey{
				Key:      secret.Name,
				Property: property,
			})
		}
	}
	return keys, nil
}

func (c *client) SetSecret(ctx context.Context, key v1alpha1.SecretKey, value []byte) error {
	// Update
	// TODO: validate if this is correct
	_, err := c.secretsClient.Patch(ctx, key.Key, types.JSONPatchType, value, metav1.PatchOptions{})
	if !errors.IsNotFound(err) {
		return err
	}

	// Create
	_, err = c.secretsClient.Create(ctx,
		&apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: key.Key,
			},
			Data: map[string][]byte{
				key.Property: value,
			},
		},
		metav1.CreateOptions{},
	)
	return err
}
