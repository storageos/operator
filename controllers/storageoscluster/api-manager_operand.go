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

// apiManagerPackage contains the resource manifests for api-manager operand.
const apiManagerPackage = "api-manager"

type APIManagerOperand struct {
	name            string
	client          client.Client
	requires        []string
	requeueStrategy operand.RequeueStrategy
	fs              filesys.FileSystem
}

var _ operand.Operand = &APIManagerOperand{}

func (c *APIManagerOperand) Name() string                             { return c.name }
func (c *APIManagerOperand) Requires() []string                       { return c.requires }
func (c *APIManagerOperand) RequeueStrategy() operand.RequeueStrategy { return c.requeueStrategy }
func (c *APIManagerOperand) ReadyCheck(ctx context.Context, obj client.Object) (bool, error) {
	return true, nil
}

func (c *APIManagerOperand) Ensure(ctx context.Context, obj client.Object, ownerRef metav1.OwnerReference) (eventv1.ReconcilerEvent, error) {
	cluster, ok := obj.(*storageoscomv1.StorageOSCluster)
	if !ok {
		return nil, fmt.Errorf("failed to convert %v to StorageOSCluster", obj)
	}

	b, err := declarative.NewBuilder(apiManagerPackage, c.fs,
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

func (c *APIManagerOperand) Delete(ctx context.Context, obj client.Object) (eventv1.ReconcilerEvent, error) {
	return nil, nil
}

func NewAPIManagerOperand(
	name string,
	client client.Client,
	requires []string,
	requeueStrategy operand.RequeueStrategy,
	fs filesys.FileSystem,
) *APIManagerOperand {
	return &APIManagerOperand{
		name:            name,
		client:          client,
		requires:        requires,
		requeueStrategy: requeueStrategy,
		fs:              fs,
	}
}
