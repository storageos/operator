package main

import (
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/darkowlzz/operator-toolkit/telemetry/export"
	"github.com/darkowlzz/operator-toolkit/webhook/cert"
	"go.uber.org/zap/zapcore"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	configstorageoscomv1 "github.com/storageos/operator/apis/config.storageos.com/v1"
	storageoscomv1 "github.com/storageos/operator/apis/v1"
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
	utilruntime.Must(configstorageoscomv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")

	var opts zap.Options
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	// Configure logger.
	f := func(ec *zapcore.EncoderConfig) {
		ec.TimeKey = "timestamp"
		ec.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	}
	encoderOpts := func(o *zap.Options) {
		o.EncoderConfigOptions = append(o.EncoderConfigOptions, f)
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts), zap.StacktraceLevel(zapcore.PanicLevel), encoderOpts))

	// Setup telemetry.
	telemetryShutdown, err := export.InstallJaegerExporter("storageos-operator")
	if err != nil {
		setupLog.Error(err, "unable to setup telemetry exporter")
		os.Exit(1)
	}
	defer telemetryShutdown()

	// Load controller manager configuration and create manager options from
	// it.
	ctrlConfig := configstorageoscomv1.OperatorConfig{}
	options := ctrl.Options{Scheme: scheme}
	if configFile != "" {
		var err error
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile).OfKind(&ctrlConfig))
		if err != nil {
			setupLog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
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
		CertRefreshInterval: ctrlConfig.WebhookCertRefreshInterval.Duration,
		Service: &admissionregistrationv1.ServiceReference{
			Name:      ctrlConfig.WebhookServiceName,
			Namespace: currentNS,
		},
		Client:                      cli,
		SecretRef:                   &types.NamespacedName{Name: ctrlConfig.WebhookSecretRef, Namespace: currentNS},
		ValidatingWebhookConfigRefs: []types.NamespacedName{{Name: ctrlConfig.ValidatingWebhookConfigRef}},
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
