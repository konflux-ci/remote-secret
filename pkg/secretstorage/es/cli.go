//
// Copyright (c) 2021 Red Hat, Inc.
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

package es

import (
	"context"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/log"

	es "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
)

func NewESSecretStorage(ctx context.Context, configFilePath string) (secretstorage.SecretStorage, error) {
	lg := log.FromContext(ctx)
	providerConf := &es.SecretStoreProvider{}
	lg.Info("open config", "config", configFilePath)
	file, err := os.Open(configFilePath) // #nosec:G304, path param and file is controlled by operator deployment
	if err != nil {
		return nil, fmt.Errorf("error opening the config file from %s: %w", configFilePath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()
	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading the config file: %w", err)
	}
	lg.Info("reading config", "content", string(bytes[:]))
	if err := yaml.Unmarshal(bytes, &providerConf); err != nil {
		return nil, fmt.Errorf("error parsing the config file as YAML: %w", err)
	}
	lg.Info("es config complete")
	return &ESStorage{ProviderConfig: providerConf}, nil
}
