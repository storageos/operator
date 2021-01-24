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
	kustomizetypes "sigs.k8s.io/kustomize/api/types"

	storageoscomv1 "github.com/storageos/operator/api/v1"
	"github.com/storageos/operator/internal/image"
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
	// Setup a tracer and start a span.
	tr := otel.Tracer("APIManagerOperand Ensure")
	ctx, span := tr.Start(ctx, "apimanageroperand ensure")
	defer span.End()

	b, err := getAPIManagerBuilder(c.fs, obj)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return nil, b.Apply(ctx)
}

func (c *APIManagerOperand) Delete(ctx context.Context, obj client.Object) (eventv1.ReconcilerEvent, error) {
	// Setup a tracer and start a span.
	tr := otel.Tracer("APIManagerOperand Delete")
	ctx, span := tr.Start(ctx, "apimanageroperand delete")
	defer span.End()

	b, err := getAPIManagerBuilder(c.fs, obj)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return nil, b.Delete(ctx)
}

func getAPIManagerBuilder(fs filesys.FileSystem, obj client.Object) (*declarative.Builder, error) {
	cluster, ok := obj.(*storageoscomv1.StorageOSCluster)
	if !ok {
		return nil, fmt.Errorf("failed to convert %v to StorageOSCluster", obj)
	}

	// Get image name.
	images := []kustomizetypes.Image{}
	namedImages := image.NamedImages{
		"api-manager": cluster.Spec.Images.APIManagerContainer,
	}
	images = append(images, image.GetKustomizeImageList(namedImages)...)

	return declarative.NewBuilder(apiManagerPackage, fs,
		declarative.WithKustomizeMutationFunc([]kustomize.MutateFunc{
			kustomize.AddNamespace(cluster.GetNamespace()),
			kustomize.AddImages(images),
		}),
	)
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
