package storageoscluster

import (
	"context"
	"fmt"

	"github.com/darkowlzz/operator-toolkit/declarative"
	"github.com/darkowlzz/operator-toolkit/declarative/kustomize"
	"github.com/darkowlzz/operator-toolkit/declarative/transform"
	eventv1 "github.com/darkowlzz/operator-toolkit/event/v1"
	"github.com/darkowlzz/operator-toolkit/operator/v1/operand"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/filesys"

	storageoscomv1 "github.com/storageos/operator/api/v1"
)

// csiPackage contains the resource manifests for csi operand.
const csiPackage = "csi"

type CSIOperand struct {
	name            string
	client          client.Client
	requires        []string
	requeueStrategy operand.RequeueStrategy
	fs              filesys.FileSystem
}

var _ operand.Operand = &CSIOperand{}

func (c *CSIOperand) Name() string                             { return c.name }
func (c *CSIOperand) Requires() []string                       { return c.requires }
func (c *CSIOperand) RequeueStrategy() operand.RequeueStrategy { return c.requeueStrategy }
func (c *CSIOperand) ReadyCheck(ctx context.Context, obj client.Object) (bool, error) {
	return true, nil
}

func (c *CSIOperand) Ensure(ctx context.Context, obj client.Object, ownerRef metav1.OwnerReference) (eventv1.ReconcilerEvent, error) {
	cluster, ok := obj.(*storageoscomv1.StorageOSCluster)
	if !ok {
		return nil, fmt.Errorf("failed to convert %v to StorageOSCluster", obj)
	}

	b, err := declarative.NewBuilder(csiPackage, c.fs,
		declarative.WithCommonTransforms([]transform.TransformFunc{
			transform.SetOwnerReference([]metav1.OwnerReference{ownerRef}),
		}),
		declarative.WithKustomizeMutationFunc([]kustomize.MutateFunc{
			kustomize.AddNamespace(cluster.GetNamespace()),
		}),
	)
	if err != nil {
		return nil, err
	}

	return nil, b.Apply(ctx)
}

func (c *CSIOperand) Delete(ctx context.Context, obj client.Object) (eventv1.ReconcilerEvent, error) {
	return nil, nil
}

func NewCSIOperand(
	name string,
	client client.Client,
	requires []string,
	requeueStrategy operand.RequeueStrategy,
	fs filesys.FileSystem,
) *CSIOperand {
	return &CSIOperand{
		name:            name,
		client:          client,
		requires:        requires,
		requeueStrategy: requeueStrategy,
		fs:              fs,
	}
}
