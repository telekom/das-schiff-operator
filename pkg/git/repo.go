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

package git

import (
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
)

type Repo interface {
	sync.Locker
	Repo() *git.Repository
	Auth() transport.AuthMethod
	DesiredRef() plumbing.ReferenceName
	DesiredRemoteRef() plumbing.ReferenceName
}

type repo struct {
	sync.Mutex
	*git.Repository
	auth   transport.AuthMethod
	branch string
}

func (r *repo) Repo() *git.Repository {
	return r.Repository
}

func (r *repo) Auth() transport.AuthMethod {
	return r.auth
}

func (r *repo) DesiredRef() plumbing.ReferenceName {
	return plumbing.NewBranchReferenceName(r.branch)
}

func (r *repo) DesiredRemoteRef() plumbing.ReferenceName {
	return plumbing.NewRemoteHEADReferenceName(r.branch)
}
