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

package controllers

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-logr/logr"
	"github.com/spf13/viper"
	"github.com/telekom/das-schiff-operator/pkg/git"
	"github.com/telekom/das-schiff-operator/pkg/sops"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha4"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubeadmControlPlaneReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	git git.Repo
}

func (r *KubeadmControlPlaneReconciler) InitGitRepository() error {

	repo, err := git.Clone(
		viper.GetString("kubeconfigBackup.repo"),
		viper.GetString("kubeconfigBackup.branch"),
		&http.BasicAuth{Password: viper.GetString("kubeconfigBackup.gitlabToken")},
		viper.GetString("kubeconfigBackup.cloneDir"),
	)
	r.git = repo
	return err
}

// +kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=kubeadmcontrolplanes,verbs=get;list;watch
// +kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=kubeadmcontrolplanes/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

func (r *KubeadmControlPlaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.git == nil {
		return fmt.Errorf("git repository not initialized")
	}
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&v1alpha4.KubeadmControlPlane{}).
		Complete(r)
}

func (r *KubeadmControlPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("KubeadmControlPlane", req.Name)
	log.Info("reconciling KCP")
	var kcp v1alpha4.KubeadmControlPlane
	if err := r.Client.Get(ctx, req.NamespacedName, &kcp); err != nil {
		// deleted objects somtimes still trigger reconciliation, we'll just ignore those.
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch KubeadmControlPlane")
		return ctrl.Result{}, err
	}

	if !kcp.Status.Initialized {
		// Kubeconfig only exists after cluster is initialized, abort.
		log.Info("Not initialized yet")
		return ctrl.Result{}, nil
	}

	var kcSecret v1.Secret
	secretName := req.NamespacedName
	secretName.Name += "-kubeconfig"
	if err := r.Client.Get(ctx, secretName, &kcSecret); err != nil {
		// A 'not found' error should not occur here, since the KCP is initialized. So no special handling.
		return ctrl.Result{}, err
	}

	kubeconfig := string(kcSecret.Data["value"])
	creationTime := kcSecret.ObjectMeta.CreationTimestamp.Time

	if err := r.addKubeconfigToRepoAndCommit(ctx, log, pathFromNamespace(req.Namespace), req.Name, kubeconfig, creationTime); err != nil {
		log.Error(err, "failed to add kubeconfig file to git")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *KubeadmControlPlaneReconciler) addKubeconfigToRepoAndCommit(ctx context.Context, log logr.Logger, dir, name, kubeconfig string, creationTime time.Time) error {
	r.git.Lock()
	defer r.git.Unlock()

	tree, err := r.git.Repo().Worktree()
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}
	if tree.PullContext(ctx, &gogit.PullOptions{Auth: r.git.Auth(), Force: true, ReferenceName: r.git.DesiredRemoteRef(), SingleBranch: true}); err != nil {
		return fmt.Errorf("failed to pull: %w", err)
	}
	if err := tree.Reset(&gogit.ResetOptions{Mode: gogit.HardReset}); err != nil {
		return fmt.Errorf("failed to reset worktree: %w", err)
	}
	err = tree.Filesystem.MkdirAll(dir, os.FileMode(0755))
	if err != nil {
		return fmt.Errorf("failed to create path: %w", err)
	}
	filePath := tree.Filesystem.Join(dir, name+".yaml")
	_, err = tree.Filesystem.Stat(filePath)
	if err == nil {
		log.V(4).Info("file already exists, checking if up to date.")
		file, err := tree.Filesystem.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		contents, err := ioutil.ReadAll(file)
		if err != nil {
			return fmt.Errorf("failed to read file contents: %w", err)
		}

		var cfg map[string]interface{}
		if err := yaml.Unmarshal(contents, &cfg); err != nil {
			return fmt.Errorf("failed to parse existing kubeconfig file: %w", err)
		}
		if sopsCfg, ok := cfg["sops"].(map[string]interface{}); ok {
			if lastModified, ok := sopsCfg["lastmodified"].(string); ok {
				lastModifiedTime, err := time.Parse(time.RFC3339Nano, lastModified)
				if err == nil && lastModifiedTime.After(creationTime) {
					log.Info("backed up file is already up to date, skipping.")
					return nil
				}
			}
		}
		log.V(4).Info("file doesn't seem to be up to date, overwriting")
	}

	file, err := tree.Filesystem.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	s, err := getSops(tree.Filesystem)
	if err != nil {
		return fmt.Errorf("failed to initialize sops: %w", err)
	}
	encrypted, err := s.EncryptYaml([]byte(kubeconfig), filePath)
	if err != nil {
		return fmt.Errorf("failed to encrypt yaml: %w", err)
	}

	_, err = file.Write(encrypted)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}
	if _, err := tree.Add(filePath); err != nil {
		return fmt.Errorf("failed to add file to tree: %w", err)
	}
	if _, err := tree.Commit("add "+filePath, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "schiff-operator",
			Email: "operator@schiff.telekom.de",
			When:  time.Now(),
		},
	}); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	if err := r.git.Repo().PushContext(ctx, &gogit.PushOptions{RemoteName: "origin", Auth: r.git.Auth()}); err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}
	return nil
}

func getSops(fs billy.Filesystem) (sops.Sops, error) {
	file, err := fs.Open("/.sops.yaml")
	if err != nil {
		return sops.Sops{}, err
	}
	contents, err := ioutil.ReadAll(file)
	if err != nil {
		return sops.Sops{}, err
	}
	return sops.NewFromConfig(contents)
}

func pathFromNamespace(namespace string) string {
	parts := strings.Split(namespace, "-")
	if len(parts) != 3 {
		return ""
	}

	return fmt.Sprintf("%s/%s", parts[2], parts[1])
}
