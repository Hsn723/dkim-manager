/*
Copyright 2022.

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
	"fmt"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/golang-jwt/jwt/v4"
	dkimmanagerv1 "github.com/hsn723/dkim-manager/api/v1"
	dkimmanagerv2 "github.com/hsn723/dkim-manager/api/v2"
	"github.com/hsn723/dkim-manager/controllers"
	"github.com/hsn723/dkim-manager/hooks"
	//+kubebuilder:scaffold:imports
)

const (
	fallbackServiceAccount = "system:serviceaccount:dkim-manager:dkim-manager-controller-manager"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(dkimmanagerv1.AddToScheme(scheme))
	utilruntime.Must(dkimmanagerv2.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func getServiceAccount() string {
	data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		setupLog.Error(err, "could not read token")
		return fallbackServiceAccount
	}
	token, _, err := jwt.NewParser().ParseUnverified(string(data), &jwt.RegisteredClaims{})
	if err != nil {
		setupLog.Error(err, "could not parse token")
		return fallbackServiceAccount
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		setupLog.Error(fmt.Errorf("invalid claims type"), "could not recognize claims")
		return fallbackServiceAccount
	}
	return claims.Subject
}

func getNamespace() string {
	data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		setupLog.Error(err, "could not get namespace")
		return ""
	}
	return string(data)
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var serviceAccount string
	var namespace string
	var namespaced bool
	var webhooksEnabled bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&serviceAccount, "service-account", getServiceAccount(), "The name of the service account.")
	flag.StringVar(&namespace, "namespace", "", "The namespace the controller should manage.")
	flag.BoolVar(&namespaced, "namespaced", false, "Only manage resources in the same namespace as the controller. The --namespace parameter, if defined, takes precedence.")
	flag.BoolVar(&webhooksEnabled, "webhooks", true, "Enable webhooks")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Client: client.Options{
			Cache: &client.CacheOptions{
				Unstructured: true,
			},
		},
		Metrics: metricsserver.Options{BindAddress: metricsAddr},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port: webhook.DefaultPort,
		}),
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "efef7e15.atelierhsn.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	dec := admission.NewDecoder(scheme)

	if namespace == "" && namespaced {
		namespace = getNamespace()
	}

	if err := (&controllers.DKIMKeyReconciler{
		Client:     mgr.GetClient(),
		Log:        ctrl.Log.WithName("controllers").WithName("DKIMKey"),
		Scheme:     mgr.GetScheme(),
		Namespace:  namespace,
		ReadClient: mgr.GetAPIReader(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DKIMKey")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if webhooksEnabled {
		hooks.SetupDKIMKeyWebhook(mgr, &dec)
		hooks.SetupDNSEndpointWebhook(mgr, &dec, serviceAccount)
		hooks.SetupSecretWebhook(mgr, &dec, serviceAccount)

		if err := ctrl.NewWebhookManagedBy(mgr, &dkimmanagerv2.DKIMKey{}).
			Complete(); err != nil {
			setupLog.Error(err, "unable to create conversion webhook", "webhook", "DKIMKey")
			os.Exit(1)
		}
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
