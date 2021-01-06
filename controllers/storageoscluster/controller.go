package storageoscluster

import (
	"context"

	compositev1 "github.com/darkowlzz/operator-toolkit/controller/composite/v1"
	"github.com/darkowlzz/operator-toolkit/object"
	operatorv1 "github.com/darkowlzz/operator-toolkit/operator/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StorageOSClusterController struct {
	Operator operatorv1.Operator
}

var _ compositev1.Controller = &StorageOSClusterController{}

func (c *StorageOSClusterController) Default(context.Context, client.Object) {}

func (c *StorageOSClusterController) Validate(context.Context, client.Object) error { return nil }

func (c *StorageOSClusterController) Initialize(context.Context, client.Object, metav1.Condition) error {
	return nil
}

func (c *StorageOSClusterController) Operate(ctx context.Context, obj client.Object) (result ctrl.Result, err error) {
	return c.Operator.Ensure(ctx, obj, object.OwnerReferenceFromObject(obj))
}

func (c *StorageOSClusterController) Cleanup(ctx context.Context, obj client.Object) (result ctrl.Result, err error) {
	return ctrl.Result{}, nil
}

func (c *StorageOSClusterController) UpdateStatus(ctx context.Context, obj client.Object) error {
	return nil
}
