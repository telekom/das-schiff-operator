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
	"os"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

func CloneInMemory(remote, branch string, auth http.AuthMethod) (Repo, error) {
	r, err := git.Clone(
		filesystem.NewStorage(memfs.New(), cache.NewObjectLRUDefault()),
		memfs.New(),
		&git.CloneOptions{
			URL:               remote,
			ReferenceName:     plumbing.NewBranchReferenceName(branch),
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
			Auth:              auth,
		})
	return &repo{Repository: r, auth: auth, branch: branch}, err
}

// Clone clones a remote repository to the local filesystem at path, but provides an in-memory workdir.
func Clone(remote, branch string, auth http.AuthMethod, path string) (Repo, error) {
	ref := plumbing.NewBranchReferenceName(branch)
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	repo := &repo{
		auth:   auth,
		branch: branch,
	}
	storage := filesystem.NewStorage(osfs.New(path), cache.NewObjectLRUDefault())
	// if there are already files in that folder, we assume that the repo is already cloned
	if len(files) > 0 {
		repo.Repository, err = git.Open(storage, memfs.New())
		if err != nil {
			return nil, err
		}
		err = repo.Repository.Fetch(&git.FetchOptions{Auth: repo.auth, Force: true})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return nil, err
		}
		tree, err := repo.Repository.Worktree()
		if err != nil {
			return nil, err
		}
		err = tree.Checkout(&git.CheckoutOptions{Branch: repo.DesiredRef(), Force: true})
		if err != nil {
			return nil, err
		}
		return repo, nil
	}
	repo.Repository, err = git.Clone(
		storage,
		memfs.New(), &git.CloneOptions{
			URL:               remote,
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
			Auth:              auth,
			ReferenceName:     ref,
		})
	return repo, err
}
