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
	"bytes"
	"context"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
	"regexp"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientv1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type client struct {
	namespace     string
	secretsClient clientv1.SecretInterface
}

func (c *client) GetSecret(ctx context.Context, key v1alpha1.SecretKey) ([]byte, error) {
	if err := validate(key); err != nil {
		return nil, err
	}
	secret, err := c.secretsClient.Get(ctx, key.GetKey(), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, v1alpha1.ErrKeyNotFound
		}
		return nil, err
	}
	return secret.Data[key.GetProperty()], nil
}

func (c *client) ListSecretKeys(ctx context.Context, query v1alpha1.SecretKeyQuery) ([]v1alpha1.SecretKey, error) {
	listData, err := c.secretsClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var keys []v1alpha1.SecretKey
	for _, secret := range listData.Items {
		for property := range secret.Data {
			key := fmt.Sprintf("%s.%s", secret.Name, property)
			if ok, _ := regexp.MatchString(query.Key.Regexp, key); ok {
				keys = append(keys, v1alpha1.SecretKey{
					Key: key,
				})
			}
		}
	}
	return keys, nil
}

func (c *client) SetSecret(ctx context.Context, key v1alpha1.SecretKey, value []byte) error {
	if err := validate(key); err != nil {
		return err
	}

	// Get
	shouldCreate := false
	secret, err := c.secretsClient.Get(ctx, key.GetKey(), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			shouldCreate = true
		}
		return err
	}

	// Create
	if shouldCreate {
		_, err = c.secretsClient.Create(ctx,
			&apiv1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: key.GetKey(),
				},
				Data: map[string][]byte{
					key.GetProperty(): value,
				},
			},
			metav1.CreateOptions{},
		)
		return err
	}

	// Update
	prop := key.GetProperty()
	if fetched, _ := secret.Data[prop]; bytes.Equal(fetched, value) {
		return nil
	}
	secret.Data[prop] = value
	_, err = c.secretsClient.Update(ctx, secret, metav1.UpdateOptions{})
	return err
}

func validate(key v1alpha1.SecretKey) error {
	if len(key.GetPath()) > 0 {
		return fmt.Errorf("kubernetes store does not accept path key")
	}
	if key.GetKey() == "" {
		return fmt.Errorf("kubernetes store requires key data")
	}
	if key.GetProperty() == "" {
		return fmt.Errorf("kubernetes store requires property data")
	}
	return nil
}
