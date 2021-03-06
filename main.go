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

package main

import (
	"flag"
	"os"

	"github.com/telekom/das-schiff-operator/pkg/config"
	"github.com/telekom/das-schiff-operator/pkg/system"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/telekom/das-schiff-operator/controllers"
	"github.com/telekom/das-schiff-operator/pkg/ipam/infoblox"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	capvV1alpha4 "sigs.k8s.io/cluster-api-provider-vsphere/api/v1alpha4"
	capiv1alpha4 "sigs.k8s.io/cluster-api/api/v1alpha4"
	cacp1alpha4 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha4"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(capvV1alpha4.AddToScheme(scheme))
	utilruntime.Must(capiv1alpha4.AddToScheme(scheme))
	utilruntime.Must(cacp1alpha4.AddToScheme(scheme))

	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var configFile string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&configFile, "config", "", "An optional path to the configuration file.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)

	setupLog.Info("starting manager", "version", system.Version)

	setupLog.Info("loading configuration", "explicit path", configFile)
	config.Load(configFile)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "114fe65d.schiff.telekom.de",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.VSphereMachineIPAMReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("VSphereMachine"),
		Scheme: mgr.GetScheme(),
		IPAM:   &infoblox.Manager{Log: logger},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VSphereMachine")
		os.Exit(1)
	}

	kubeadmControlPlaneReconciler := &controllers.KubeadmControlPlaneReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("KubeadmControlPlane"),
		Scheme: mgr.GetScheme(),
	}
	if err = kubeadmControlPlaneReconciler.InitGitRepository(); err != nil {
		setupLog.Error(err, "failed to setup git repository for kubeadmin backup", "controller", "KubeadmControlPlane")
		os.Exit(1)
	}
	if err = kubeadmControlPlaneReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VSphereMachine")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
