package storageoscluster

import (
	"context"
	"fmt"

	"github.com/darkowlzz/operator-toolkit/declarative"
	"github.com/darkowlzz/operator-toolkit/declarative/kustomize"
	eventv1 "github.com/darkowlzz/operator-toolkit/event/v1"
	"github.com/darkowlzz/operator-toolkit/operator/v1/operand"
	"go.opentelemetry.io/otel"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/filesys"

	storageoscomv1 "github.com/storageos/operator/api/v1"
)

// nodePackage contains the resource manifests for node operand.
const nodePackage = "node"

type NodeOperand struct {
	name            string
	client          client.Client
	requires        []string
	requeueStrategy operand.RequeueStrategy
	fs              filesys.FileSystem
}

var _ operand.Operand = &NodeOperand{}

func (c *NodeOperand) Name() string                             { return c.name }
func (c *NodeOperand) Requires() []string                       { return c.requires }
func (c *NodeOperand) RequeueStrategy() operand.RequeueStrategy { return c.requeueStrategy }
func (c *NodeOperand) ReadyCheck(ctx context.Context, obj client.Object) (bool, error) {
	return true, nil
}

func (c *NodeOperand) Ensure(ctx context.Context, obj client.Object, ownerRef metav1.OwnerReference) (eventv1.ReconcilerEvent, error) {
	// Setup a tracer and start a span.
	tr := otel.Tracer("NodeOperand Ensure")
	ctx, span := tr.Start(ctx, "nodeoperand ensure")
	defer span.End()

	b, err := getNodeBuilder(c.fs, obj)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return nil, b.Apply(ctx)
}

func (c *NodeOperand) Delete(ctx context.Context, obj client.Object) (eventv1.ReconcilerEvent, error) {
	// Setup a tracer and start a span.
	tr := otel.Tracer("NodeOperand Delete")
	ctx, span := tr.Start(ctx, "nodeoperand delete")
	defer span.End()

	b, err := getNodeBuilder(c.fs, obj)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return nil, b.Delete(ctx)
}

func getNodeBuilder(fs filesys.FileSystem, obj client.Object) (*declarative.Builder, error) {
	cluster, ok := obj.(*storageoscomv1.StorageOSCluster)
	if !ok {
		return nil, fmt.Errorf("failed to convert %v to StorageOSCluster", obj)
	}

	return declarative.NewBuilder(nodePackage, fs,
		declarative.WithKustomizeMutationFunc([]kustomize.MutateFunc{
			kustomize.AddNamespace(cluster.GetNamespace()),
		}),
	)
}

func NewNodeOperand(
	name string,
	client client.Client,
	requires []string,
	requeueStrategy operand.RequeueStrategy,
	fs filesys.FileSystem,
) *NodeOperand {
	return &NodeOperand{
		name:            name,
		client:          client,
		requires:        requires,
		requeueStrategy: requeueStrategy,
		fs:              fs,
	}
}
