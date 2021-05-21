package controllers

import (
	"context"
	"fmt"

	compositev1 "github.com/darkowlzz/operator-toolkit/controller/composite/v1"
	"github.com/darkowlzz/operator-toolkit/declarative/loader"
	"github.com/darkowlzz/operator-toolkit/operator/v1/executor"
	"github.com/darkowlzz/operator-toolkit/telemetry"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storageoscomv1 "github.com/storageos/operator/api/v1"
	"github.com/storageos/operator/controllers/storageoscluster"
)

const instrumentationName = "github.com/storageos/operator/controllers"

var instrumentation *telemetry.Instrumentation

func init() {
	// Setup package instrumentation.
	instrumentation = telemetry.NewInstrumentationWithProviders(
		instrumentationName, nil, nil,
		ctrl.Log.WithName("controllers").WithName("StorageOSCluster"),
	)
}

// StorageOSClusterReconciler reconciles a StorageOSCluster object
type StorageOSClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	compositev1.CompositeReconciler
}

func NewStorageOSClusterReconciler(mgr ctrl.Manager) *StorageOSClusterReconciler {
	return &StorageOSClusterReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
}

// +kubebuilder:rbac:groups=storageos.com,resources=storageosclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=storageos.com,resources=storageosclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=storageos.com,resources=storageosclusters/finalizers,verbs=update

// TODO: Remove this. Temporary, for initial development only.
// +kubebuilder:rbac:groups=*,resources=*,verbs=*

// SetupWithManager sets up the controller with the Manager.
func (r *StorageOSClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	_, span, _, log := instrumentation.Start(context.Background(), "StorageosCluster.SetupWithManager")
	defer span.End()

	// Load manifests in an in-memory filesystem.
	fs, err := loader.NewLoadedManifestFileSystem("channels", "stable")
	if err != nil {
		return fmt.Errorf("failed to create loaded ManifestFileSystem: %w", err)
	}

	// TODO: Expose the executor strategy option via SetupWithManager.
	cc, err := storageoscluster.NewStorageOSClusterController(mgr, fs, executor.Parallel)
	if err != nil {
		return err
	}

	// Initialize the reconciler.
	err = r.CompositeReconciler.Init(mgr, cc, &storageoscomv1.StorageOSCluster{},
		compositev1.WithName("storageoscluster-controller"),
		compositev1.WithCleanupStrategy(compositev1.FinalizerCleanup),
		compositev1.WithInitCondition(compositev1.DefaultInitCondition),
		compositev1.WithInstrumentation(nil, nil, log),
	)
	if err != nil {
		return fmt.Errorf("failed to create new CompositeReconciler: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&storageoscomv1.StorageOSCluster{}).
		Complete(r)
}
