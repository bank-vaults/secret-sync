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
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type client struct {
	namespace     string
	secretsClient corev1.SecretInterface
}

func (c *client) GetSecret(ctx context.Context, key v1alpha1.SecretKey) ([]byte, error) {
	secret, err := c.secretsClient.Get(ctx, key.Key, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	fmt.Println("get k8s", secret.Data)
	return nil, nil
}

func (c *client) ListSecretKeys(ctx context.Context, query v1alpha1.SecretKeyQuery) ([]v1alpha1.SecretKey, error) {
	secret, err := c.secretsClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	fmt.Println("list k8s", secret.Items)
	return nil, nil
}

func (c *client) SetSecret(ctx context.Context, key v1alpha1.SecretKey, value []byte) error {
	secret, err := c.secretsClient.Get(ctx, key.Key, metav1.GetOptions{})
	if err != nil {
		return err
	}
	updatedSecret, err := c.secretsClient.Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	fmt.Println("set original k8s", secret.Data)
	fmt.Println("set updated k8s", updatedSecret.Data)
	return nil
}
