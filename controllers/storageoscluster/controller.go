package storageoscluster

import (
	"context"
	"fmt"

	compositev1 "github.com/darkowlzz/operator-toolkit/controller/composite/v1"
	"github.com/darkowlzz/operator-toolkit/object"
	operatorv1 "github.com/darkowlzz/operator-toolkit/operator/v1"
	storageoscomv1 "github.com/storageos/operator/api/v1"
	"go.opentelemetry.io/otel"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StorageOSClusterController struct {
	Operator operatorv1.Operator
	Client   client.Client
}

var _ compositev1.Controller = &StorageOSClusterController{}

func (c *StorageOSClusterController) Default(context.Context, client.Object) {}

func (c *StorageOSClusterController) Validate(context.Context, client.Object) error { return nil }

func (c *StorageOSClusterController) Initialize(ctx context.Context, obj client.Object, condn metav1.Condition) error {
	tr := otel.Tracer("Initialize")
	_, span := tr.Start(ctx, "initialization")
	defer span.End()

	cluster, ok := obj.(*storageoscomv1.StorageOSCluster)
	if !ok {
		return fmt.Errorf("failed to convert %v to StorageOSCluster", obj)
	}

	meta.SetStatusCondition(&cluster.Status.Conditions, condn)

	return nil
}

func (c *StorageOSClusterController) Operate(ctx context.Context, obj client.Object) (result ctrl.Result, err error) {
	return c.Operator.Ensure(ctx, obj, object.OwnerReferenceFromObject(obj))
}

func (c *StorageOSClusterController) Cleanup(ctx context.Context, obj client.Object) (result ctrl.Result, err error) {
	return c.Operator.Cleanup(ctx, obj)
}

func (c *StorageOSClusterController) UpdateStatus(ctx context.Context, obj client.Object) error {
	tr := otel.Tracer("UpdateStatus")
	ctx, span := tr.Start(ctx, "update status")
	defer span.End()

	cluster, ok := obj.(*storageoscomv1.StorageOSCluster)
	if !ok {
		return fmt.Errorf("failed to convert %v to StorageOSCluster", obj)
	}

	// Gather status of the world and set in cluster status.

	if getErr := c.Client.Get(ctx, client.ObjectKeyFromObject(cluster), cluster); getErr != nil {
		if apierrors.IsNotFound(getErr) {
			// If the object is not found, the object may have been deleted by
			// cleanup handler call before UpdateStatus was called.
			return nil
		}
		return fmt.Errorf("failed to get StorageOSCluster %q: %w", cluster.GetName(), getErr)
	}

	// Check status of all the components.
	ready := true

	if ready {
		// Remove progressing condition and set cluster Ready status.
		meta.RemoveStatusCondition(&cluster.Status.Conditions, "Progressing")
		readyCondition := metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionTrue,
			Reason:  "Ready",
			Message: "Cluster Ready",
		}
		meta.SetStatusCondition(&cluster.Status.Conditions, readyCondition)
	}

	return nil
}
