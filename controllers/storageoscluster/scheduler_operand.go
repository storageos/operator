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

// schedulerPackage contains the resource manifests for scheduler operand.
const schedulerPackage = "scheduler"

type SchedulerOperand struct {
	name            string
	client          client.Client
	requires        []string
	requeueStrategy operand.RequeueStrategy
	fs              filesys.FileSystem
}

var _ operand.Operand = &SchedulerOperand{}

func (c *SchedulerOperand) Name() string                             { return c.name }
func (c *SchedulerOperand) Requires() []string                       { return c.requires }
func (c *SchedulerOperand) RequeueStrategy() operand.RequeueStrategy { return c.requeueStrategy }
func (c *SchedulerOperand) ReadyCheck(ctx context.Context, obj client.Object) (bool, error) {
	return true, nil
}

func (c *SchedulerOperand) Ensure(ctx context.Context, obj client.Object, ownerRef metav1.OwnerReference) (eventv1.ReconcilerEvent, error) {
	cluster, ok := obj.(*storageoscomv1.StorageOSCluster)
	if !ok {
		return nil, fmt.Errorf("failed to convert %v to StorageOSCluster", obj)
	}

	b, err := declarative.NewBuilder(schedulerPackage, c.fs,
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

func (c *SchedulerOperand) Delete(ctx context.Context, obj client.Object) (eventv1.ReconcilerEvent, error) {
	return nil, nil
}

func NewSchedulerOperand(
	name string,
	client client.Client,
	requires []string,
	requeueStrategy operand.RequeueStrategy,
	fs filesys.FileSystem,
) *SchedulerOperand {
	return &SchedulerOperand{
		name:            name,
		client:          client,
		requires:        requires,
		requeueStrategy: requeueStrategy,
		fs:              fs,
	}
}
