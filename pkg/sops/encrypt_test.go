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
	"io/fs"
	"os"
	"path"
	"testing"

	"filippo.io/age"
	. "github.com/onsi/gomega"
	"go.mozilla.org/sops/v3/decrypt"
)

const ()

func TestEncryptYaml(t *testing.T) {
	g := NewWithT(t)

	identity, err := age.GenerateX25519Identity()
	g.Expect(err).To(BeNil())

	pubKey := identity.Recipient().String()
	delkey, err := ageKeyToEnv(identity)
	g.Expect(err).To(BeNil())
	defer delkey()

	s := Sops{
		cfg: configFile{
			CreationRules: []creationRule{
				{Age: pubKey, EncryptedRegex: "^secret$"},
			},
		},
	}
	plain := []byte("hello: world\nsecret: society\n")
	enc, err := s.EncryptYaml(plain, "test.yaml")
	g.Expect(err).To(BeNil())
	g.Expect(enc).NotTo(Equal(plain))
	fmt.Println(string(enc))

	clear, err := decrypt.Data(enc, "yaml")
	g.Expect(err).To(BeNil())
	g.Expect(clear).To(Equal(plain))
}

func ageKeyToEnv(identity *age.X25519Identity) (func(), error) {
	tmpDir, err := os.MkdirTemp("", "schiff-operator-test")
	if err != nil {
		return nil, err
	}
	path := path.Join(tmpDir, "sops.txt")
	if err := os.WriteFile(path, []byte(identity.String()), fs.FileMode(0700)); err != nil {
		return nil, err
	}

	if err := os.Setenv("SOPS_AGE_KEY_FILE", path); err != nil {
		return nil, err
	}
	return func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			panic(err)
		}
	}, nil
}
