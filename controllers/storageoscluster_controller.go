package controllers

import (
	"fmt"

	compositev1 "github.com/darkowlzz/operator-toolkit/controller/composite/v1"
	"github.com/darkowlzz/operator-toolkit/declarative/loader"
	"github.com/darkowlzz/operator-toolkit/operator/v1/executor"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storageoscomv1 "github.com/storageos/operator/api/v1"
	"github.com/storageos/operator/controllers/storageoscluster"
)

// StorageOSClusterReconciler reconciles a StorageOSCluster object
type StorageOSClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	compositev1.CompositeReconciler
}

// +kubebuilder:rbac:groups=storageos.com,resources=storageosclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=storageos.com,resources=storageosclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=storageos.com,resources=storageosclusters/finalizers,verbs=update

// SetupWithManager sets up the controller with the Manager.
func (r *StorageOSClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
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
	err = r.CompositeReconciler.Init(mgr, &storageoscomv1.StorageOSCluster{},
		compositev1.WithName("storageoscluster-controller"),
		compositev1.WithController(cc),
		compositev1.WithCleanupStrategy(compositev1.FinalizerCleanup),
		compositev1.WithInitCondition(compositev1.DefaultInitCondition),
		compositev1.WithLogger(r.Log),
	)
	if err != nil {
		return fmt.Errorf("failed to create new CompositeReconciler: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&storageoscomv1.StorageOSCluster{}).
		Complete(r)
}
