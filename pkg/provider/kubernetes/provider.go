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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Provider struct{}

func (p *Provider) NewClient(_ context.Context, backend v1alpha1.SecretStoreProvider) (v1alpha1.StoreClient, error) {
	providerCfg := backend.Kubernetes
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", providerCfg.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kube config: %w", err)
	}
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build kube client: %w", err)
	}

	return &client{
		namespace:     providerCfg.Namespace,
		secretsClient: kubeClient.CoreV1().Secrets(providerCfg.Namespace),
	}, nil
}

func (p *Provider) Validate(backend v1alpha1.SecretStoreProvider) error {
	providerCfg := backend.Kubernetes
	if providerCfg == nil {
		return fmt.Errorf("empty Kubernetes config")
	}
	if providerCfg.ConfigPath == "" {
		return fmt.Errorf("empty .Kubernetes.ConfigPath")
	}
	if providerCfg.Namespace == "" {
		return fmt.Errorf("empty .Kubernetes.Namespace")
	}
	return nil
}

func init() {
	v1alpha1.Register(&Provider{}, &v1alpha1.SecretStoreProvider{
		Kubernetes: &v1alpha1.SecretStoreProviderKubernetes{},
	})
}
