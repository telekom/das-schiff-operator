/*
Copyright 2021 Deutsche Telekom AG.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sops

import (
	"fmt"

	"go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/aes"
	"go.mozilla.org/sops/v3/cmd/sops/common"
	"go.mozilla.org/sops/v3/stores/yaml"
	"go.mozilla.org/sops/v3/version"
)

var cipher = aes.NewCipher()

var store = &yaml.Store{}

type Sops struct {
	cfg configFile
}

func NewFromConfig(configBytes []byte) (Sops, error) {
	conf, err := configFromFile(configBytes)
	return Sops{
		cfg: conf,
	}, err
}

func (s *Sops) EncryptYaml(b []byte, path string) ([]byte, error) {
	cfg, err := FileConfig(s.cfg, path)
	if err != nil {
		return nil, err
	}

	branches, err := store.LoadPlainFile(b)
	if err != nil {
		return nil, err
	}
	tree := sops.Tree{
		Branches: branches,
		Metadata: sops.Metadata{
			KeyGroups:      cfg.KeyGroups,
			EncryptedRegex: cfg.EncryptedRegex,
			Version:        version.Version,
		},
	}
	dataKey, errs := tree.GenerateDataKey()
	if len(errs) > 0 {
		return nil, fmt.Errorf("could not generate data key: %v", errs)
	}

	if err = common.EncryptTree(common.EncryptTreeOpts{
		DataKey: dataKey,
		Tree:    &tree,
		Cipher:  cipher,
	}); err != nil {
		return nil, err
	}

	return store.EmitEncryptedFile(tree)
}
