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

package storesync

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"text/template"

	"github.com/iancoleman/strcase"
	"golang.org/x/sync/errgroup"

	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
)

type fetchResponse struct {
	// Always set
	Data []byte

	// Only 1 is non-nil
	FromRef    *v1alpha1.SecretRef
	FromQuery  *v1alpha1.SecretQuery
	FromSource *v1alpha1.SecretSource
}

type syncRequest struct {
	RequestID int
	ActionRef *v1alpha1.SyncAction
	Data      []byte
}

// processor is used to optimally fetch secrets from a source to internal fetched map.
type processor struct {
	mu      sync.RWMutex
	source  v1alpha1.StoreReader
	fetched map[v1alpha1.SecretRef][]byte
}

func newProcessor(source v1alpha1.StoreReader) *processor {
	return &processor{
		mu:      sync.RWMutex{},
		source:  source,
		fetched: map[v1alpha1.SecretRef][]byte{},
	}
}

// GetSyncRequests fetches the data from source and applies templating based on the provided v1alpha1.SyncAction.
// Returned map defines all secrets that need to be sent to the target store to complete the request.
func (p *processor) GetSyncRequests(ctx context.Context, reqID int, req v1alpha1.SyncAction) (map[v1alpha1.SecretRef]syncRequest, error) {
	switch {
	// FromRef can only sync a single secret
	case req.FromRef != nil:
		resp, err := p.FetchFromRef(ctx, *req.FromRef)
		if err != nil {
			return nil, err
		}

		syncRef := *req.FromRef
		if req.Target.Key != nil {
			syncRef.Key = *req.Target.Key
		}

		syncValue := resp.Data
		if !isTemplateEmpty(req.Template) {
			syncValue, err = getTemplatedValue(req.Template, string(resp.Data))
			if err != nil {
				return nil, err
			}
		}

		return map[v1alpha1.SecretRef]syncRequest{
			syncRef: {
				Data:      syncValue,
				ActionRef: &req,
				RequestID: reqID,
			},
		}, nil

	// FromQuery can sync both a single secret or multiple secrets
	case req.FromQuery != nil:
		fetchResps, err := p.FetchFromQuery(ctx, *req.FromQuery)
		if err != nil {
			return nil, err
		}

		// Handle FromQuery => Key
		if req.Target.Key != nil {
			if req.Flatten == nil || !*req.Flatten {
				return nil, fmt.Errorf("requires 'flatten' for 'fromQuery' and 'target.key'")
			}

			syncRef := v1alpha1.SecretRef{
				Key:     *req.Target.Key,
				Version: nil,
			}

			templateData := make(map[string]string)
			for ref, resp := range fetchResps {
				keyName := strcase.ToLowerCamel(ref.GetName())
				templateData[keyName] = string(resp.Data)
			}
			if isTemplateEmpty(req.Template) {
				return nil, fmt.Errorf("requires 'template' for 'fromQuery' and 'target.key'")
			}
			syncValue, err := getTemplatedValue(req.Template, templateData)
			if err != nil {
				return nil, err
			}

			return map[v1alpha1.SecretRef]syncRequest{
				syncRef: {
					Data:      syncValue,
					ActionRef: &req,
					RequestID: reqID,
				},
			}, nil
		}

		// Handle FromQuery => KeyPrefix or empty
		if req.Flatten != nil && *req.Flatten {
			return nil, fmt.Errorf("cannot use 'flatten' for 'fromQuery' and 'target.key'")
		}

		syncMap := make(map[v1alpha1.SecretRef]syncRequest)
		for ref, resp := range fetchResps {
			syncRef := ref
			if req.Target.KeyPrefix != nil {
				syncRef.Key = *req.Target.KeyPrefix + ref.GetName()
			}

			syncValue := resp.Data
			if !isTemplateEmpty(req.Template) {
				syncValue, err = getTemplatedValue(req.Template, string(resp.Data))
				if err != nil {
					return nil, err
				}
			}

			syncMap[syncRef] = syncRequest{
				Data:      syncValue,
				ActionRef: &req,
				RequestID: reqID,
			}
		}
		return syncMap, nil

	// FromSources can only sync a single secret
	case len(req.FromSources) > 0:
		fetchResps, err := p.FetchFromSources(ctx, req.FromSources)
		if err != nil {
			return nil, err
		}

		if req.Target.Key == nil {
			return nil, fmt.Errorf("requires 'target.key' for 'fromSources'")
		}
		syncRef := v1alpha1.SecretRef{
			Key:     *req.Target.Key,
			Version: nil,
		}

		templateData := make(map[string]interface{})
		for ref, resp := range fetchResps {
			// For responses originating fromRef
			source := resp.FromSource
			sourceName := strcase.ToLowerCamel(source.Name)
			if source.FromRef != nil {
				// Ensures that .Data.<SOURCE NAME> fromRef is the secret value
				templateData[sourceName] = string(resp.Data)
			}
			if source.FromQuery != nil {
				// ensures that .Data.<SOURCE NAME>.<QUERY KEY> fromQuery is the secret value
				if templateData[sourceName] == nil {
					templateData[sourceName] = make(map[string]string)
				}
				keyName := strcase.ToLowerCamel(ref.GetName())
				templateData[sourceName].(map[string]string)[keyName] = string(resp.Data)
			}
		}
		if isTemplateEmpty(req.Template) {
			return nil, fmt.Errorf("requires 'template' for 'fromSources'")
		}
		syncValue, err := getTemplatedValue(req.Template, templateData)
		if err != nil {
			return nil, err
		}

		return map[v1alpha1.SecretRef]syncRequest{
			syncRef: {
				Data:      syncValue,
				ActionRef: &req,
				RequestID: reqID,
			},
		}, nil
	}

	return nil, fmt.Errorf("no sources specified")
}

// FetchFromRef fetches v1alpha1.SecretRef data from reference or from internal fetch store.
func (p *processor) FetchFromRef(ctx context.Context, fromRef v1alpha1.SecretRef) (*fetchResponse, error) {
	// Get from fetch store
	data, exists := p.getFetchedSecret(fromRef)

	// Fetch and save if not found
	if !exists {
		var err error
		data, err = p.source.GetSecret(ctx, fromRef)
		if err != nil {
			return nil, err
		}
		p.addFetchedSecret(fromRef, data)
	}

	// Return
	return &fetchResponse{
		Data:    data,
		FromRef: &fromRef,
	}, nil
}

// FetchFromQuery fetches v1alpha1.SecretRef data from query or from internal fetch store.
func (p *processor) FetchFromQuery(ctx context.Context, fromQuery v1alpha1.SecretQuery) (map[v1alpha1.SecretRef]fetchResponse, error) {
	// List secrets from source
	keyRefs, err := p.source.ListSecretKeys(ctx, fromQuery)
	if err != nil {
		return nil, fmt.Errorf("failed while doing query %v: %w", fromQuery, err)
	}

	// Fetch queried keys in parallel
	fetchMu := sync.Mutex{}
	fetched := make(map[v1alpha1.SecretRef]fetchResponse)
	fetchGroup, fetchCtx := errgroup.WithContext(ctx)
	for _, ref := range keyRefs {
		func(ref v1alpha1.SecretRef) {
			fetchGroup.Go(func() error {
				// Fetch
				resp, err := p.FetchFromRef(fetchCtx, ref)
				if err != nil {
					return err
				}

				// Update
				fetchMu.Lock()
				fetched[ref] = fetchResponse{
					Data:      resp.Data,
					FromQuery: &fromQuery,
				}
				fetchMu.Unlock()
				return nil
			})
		}(ref)
	}

	// Return
	if err = fetchGroup.Wait(); err != nil {
		return nil, err
	}
	return fetched, nil
}

// FetchFromSources fetches v1alpha1.SecretRef data from selectors or from internal fetch store..
func (p *processor) FetchFromSources(ctx context.Context, fromSources []v1alpha1.SecretSource) (map[v1alpha1.SecretRef]fetchResponse, error) {
	// Fetch source keys from source or fetch store in parallel
	fetchMu := sync.Mutex{}
	fetched := make(map[v1alpha1.SecretRef]fetchResponse)
	fetchGroup, fetchCtx := errgroup.WithContext(ctx)
	for _, src := range fromSources {
		func(src v1alpha1.SecretSource) {
			fetchGroup.Go(func() error {
				// Fetch
				kvData := make(map[v1alpha1.SecretRef][]byte)
				switch {
				case src.FromRef != nil:
					resp, err := p.FetchFromRef(fetchCtx, *src.FromRef)
					if err != nil {
						return err
					}
					kvData[*src.FromRef] = resp.Data

				case src.FromQuery != nil:
					respMap, err := p.FetchFromQuery(fetchCtx, *src.FromQuery)
					if err != nil {
						return err
					}
					for ref, resp := range respMap {
						kvData[ref] = resp.Data
					}

				default:
					return fmt.Errorf("both ref and query are empty")
				}

				// Update
				fetchMu.Lock()
				for ref, value := range kvData {
					fetched[ref] = fetchResponse{
						Data:       value,
						FromSource: &src,
					}
				}
				fetchMu.Unlock()
				return nil
			})
		}(src)
	}

	// Return
	if err := fetchGroup.Wait(); err != nil {
		return nil, err
	}
	return fetched, nil
}

// getFetchedSecret returns a key value from local fetched source.
func (p *processor) getFetchedSecret(ref v1alpha1.SecretRef) ([]byte, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	res, ok := p.fetched[ref]
	return res, ok
}

// addFetchedSecret adds a key value to local fetched store.
func (p *processor) addFetchedSecret(ref v1alpha1.SecretRef, value []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.fetched[ref] = value
}

func getTemplatedValue(syncTemplate *v1alpha1.SyncTemplate, templateData interface{}) ([]byte, error) {
	templater := template.New("template").Funcs(getTemplateFuncs())

	// Handle Template.RawData
	if syncTemplate.RawData != nil {
		templater, err := templater.Parse(*syncTemplate.RawData)
		if err != nil {
			return nil, err
		}
		output := new(bytes.Buffer)
		if err = templater.Execute(output, struct{ Data interface{} }{Data: templateData}); err != nil {
			return nil, err
		}
		return output.Bytes(), nil
	}

	// Handle Template.Data
	if len(syncTemplate.Data) > 0 {
		outputMap := make(map[string]string)
		for key, keyTpl := range syncTemplate.Data {
			templater, err := templater.Parse(keyTpl)
			if err != nil {
				return nil, err
			}
			output := new(bytes.Buffer)
			if err = templater.Execute(output, struct{ Data interface{} }{Data: templateData}); err != nil {
				return nil, err
			}
			outputMap[key] = output.String()
		}

		return json.Marshal(outputMap)
	}

	return nil, fmt.Errorf("cannot apply empty template")
}

// isTemplateEmpty checks if template is defined.
// TODO: debug why syncTemplate is sometimes not nil when not specified
func isTemplateEmpty(syncTemplate *v1alpha1.SyncTemplate) bool {
	if syncTemplate == nil {
		return true
	}
	return syncTemplate.RawData == nil && len(syncTemplate.Data) == 0
}

func getTemplateFuncs() map[string]any {
	return map[string]any{
		"base64dec": func(encoded string) string {
			decoded, _ := base64.StdEncoding.DecodeString(encoded)
			return string(decoded)
		},
		"base64enc": func(decoded string) string {
			encoded := base64.StdEncoding.EncodeToString([]byte(decoded))
			return encoded
		},
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
	}
}
