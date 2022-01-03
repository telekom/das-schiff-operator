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
	"regexp"

	"go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/age"
	"go.mozilla.org/sops/v3/config"
	"gopkg.in/yaml.v3"
)

func configFromFile(file []byte) (configFile, error) {
	cf := configFile{}
	err := yaml.Unmarshal(file, &cf)
	return cf, err
}

func FileConfig(cf configFile, path string) (config.Config, error) {
	// If config file doesn't contain CreationRules (it's empty or only contains DestionationRules), assume it does not exist
	if cf.CreationRules == nil {
		return config.Config{}, nil
	}

	var rule *creationRule

	for _, r := range cf.CreationRules {
		if r.PathRegex == "" {
			rule = &r
			break
		}
		reg, err := regexp.Compile(r.PathRegex)
		if err != nil {
			return config.Config{}, fmt.Errorf("can not compile regexp: %w", err)
		}
		if reg.MatchString(path) {
			rule = &r
			break
		}
	}

	if rule == nil {
		return config.Config{}, fmt.Errorf("error loading config: no matching creation rules found")
	}

	return configFromRule(rule)
}

func configFromRule(rule *creationRule) (config.Config, error) {
	groups, err := getKeyGroupsFromCreationRule(rule)

	return config.Config{
		KeyGroups:      groups,
		EncryptedRegex: rule.EncryptedRegex,
	}, err
}

// stripped from sops, omitting KMS support
func getKeyGroupsFromCreationRule(cRule *creationRule) ([]sops.KeyGroup, error) {
	var groups []sops.KeyGroup
	if len(cRule.KeyGroups) > 0 {
		for _, group := range cRule.KeyGroups {
			var keyGroup sops.KeyGroup
			for _, k := range group.Age {
				key, err := age.MasterKeyFromRecipient(k)
				if err != nil {
					return nil, err
				}
				keyGroup = append(keyGroup, key)
			}
			groups = append(groups, keyGroup)
		}
	} else {
		var keyGroup sops.KeyGroup
		if cRule.Age != "" {
			ageKeys, err := age.MasterKeysFromRecipients(cRule.Age)
			if err != nil {
				return nil, err
			} else {
				for _, ak := range ageKeys {
					keyGroup = append(keyGroup, ak)
				}
			}
		}
		groups = append(groups, keyGroup)
	}
	return groups, nil
}

type configFile struct {
	CreationRules []creationRule `yaml:"creation_rules"`
}

type keyGroup struct {
	Age []string `yaml:"age"`
	// PGP     []string
}

type creationRule struct {
	PathRegex string `yaml:"path_regex"`
	Age       string `yaml:"age"`
	// PGP               string
	KeyGroups       []keyGroup `yaml:"key_groups"`
	ShamirThreshold int        `yaml:"shamir_threshold"`
	// UnencryptedSuffix string     `yaml:"unencrypted_suffix"`
	// EncryptedSuffix   string     `yaml:"encrypted_suffix"`
	// UnencryptedRegex  string     `yaml:"unencrypted_regex"`
	EncryptedRegex string `yaml:"encrypted_regex"`
}
