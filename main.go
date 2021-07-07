package main

import (
	"flag"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/darkowlzz/operator-toolkit/telemetry/export"
	"github.com/darkowlzz/operator-toolkit/webhook/cert"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	storageoscomv1 "github.com/storageos/operator/api/v1"
	"github.com/storageos/operator/controllers"
	whctrlr "github.com/storageos/operator/controllers/webhook"
	// +kubebuilder:scaffold:imports
)

// podNamespace is the operator's pod namespace environment variable.
const podNamespace = "POD_NAMESPACE"

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(storageoscomv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Setup telemetry.
	telemetryShutdown, err := export.InstallJaegerExporter("storageos-operator")
	if err != nil {
		setupLog.Error(err, "unable to setup telemetry exporter")
		os.Exit(1)
	}
	defer telemetryShutdown()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "storageos-operator-lease",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Create an uncached client to be used in the certificate manager.
	// NOTE: Cached client from manager can't be used here because the cache is
	// uninitialized at this point.
	cli, err := client.New(mgr.GetConfig(), client.Options{Scheme: mgr.GetScheme()})
	if err != nil {
		setupLog.Error(err, "failed to create raw client")
		os.Exit(1)
	}

	currentNS := os.Getenv(podNamespace)
	if len(currentNS) == 0 {
		setupLog.Error(err, "failed to get current namespace")
		os.Exit(1)
	}

	// Configure the certificate manager.
	certOpts := cert.Options{
		CertRefreshInterval: 15 * time.Minute,
		Service: &admissionregistrationv1.ServiceReference{
			Name:      "storageos-operator-webhook-service",
			Namespace: currentNS,
		},
		Client:                      cli,
		SecretRef:                   &types.NamespacedName{Name: "storageos-operator-webhook-secret", Namespace: currentNS},
		ValidatingWebhookConfigRefs: []types.NamespacedName{{Name: "storageos-operator-validating-webhook-configuration"}},
	}
	// Create certificate manager without manager to start the provisioning
	// immediately.
	// NOTE: Certificate Manager implements nonLeaderElectionRunnable interface
	// but since the webhook server is also a nonLeaderElectionRunnable, they
	// start at the same time, resulting in a race condition where sometimes
	// the certificates aren't available when the webhook server starts. By
	// passing nil instead of the manager, the certificate manager is not
	// managed by the controller manager. It starts immediately, in a blocking
	// fashion, ensuring that the cert is created before the webhook server
	// starts.
	if err := cert.NewManager(nil, certOpts); err != nil {
		setupLog.Error(err, "unable to provision certificate")
		os.Exit(1)
	}

	if err = controllers.NewStorageOSClusterReconciler(mgr).
		SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller",
			"controller", "StorageOSCluster")
		os.Exit(1)
	}

	// Create and set up admission webhook controller.
	clusterWh, err := whctrlr.NewStorageOSClusterWebhook(mgr.GetClient(), mgr.GetScheme())
	if err != nil {
		setupLog.Error(err, "unable to create admission webhook controller",
			"controller", clusterWh.CtrlName)
		os.Exit(1)
	}
	if err := clusterWh.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to setup webhook controller with manager",
			"controller", clusterWh.CtrlName)
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

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
