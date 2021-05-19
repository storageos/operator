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
	kustomizetypes "sigs.k8s.io/kustomize/api/types"

	storageoscomv1 "github.com/storageos/operator/api/v1"
	"github.com/storageos/operator/internal/image"
	stransform "github.com/storageos/operator/internal/transform"
)

const (
	// schedulerPackage contains the resource manifests for scheduler operand.
	schedulerPackage = "scheduler"

	// schedulerContainer is the name of the scheduler container.
	schedulerContainer = "storageos-scheduler"
)

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
	ctx, span, _, _ := instrumentation.Start(ctx, "SchedulerOperand.Ensure")
	defer span.End()

	b, err := getSchedulerBuilder(c.fs, obj)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return nil, b.Apply(ctx)
}

func (c *SchedulerOperand) Delete(ctx context.Context, obj client.Object) (eventv1.ReconcilerEvent, error) {
	ctx, span, _, _ := instrumentation.Start(ctx, "SchedulerOperand.Delete")
	defer span.End()

	b, err := getSchedulerBuilder(c.fs, obj)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return nil, b.Delete(ctx)
}

func getSchedulerBuilder(fs filesys.FileSystem, obj client.Object) (*declarative.Builder, error) {
	cluster, ok := obj.(*storageoscomv1.StorageOSCluster)
	if !ok {
		return nil, fmt.Errorf("failed to convert %v to StorageOSCluster", obj)
	}

	// Get image name.
	images := []kustomizetypes.Image{}
	namedImages := image.NamedImages{
		"kube-scheduler": cluster.Spec.Images.KubeSchedulerContainer,
	}
	images = append(images, image.GetKustomizeImageList(namedImages)...)

	// Create deployment transforms.
	deploymentTransforms := []transform.TransformFunc{}

	// Add container args.
	argsTF := stransform.AppendPodTemplateContainerArgsFunc(schedulerContainer,
		[]string{
			fmt.Sprintf("--policy-configmap-namespace=%s", cluster.Namespace),
			fmt.Sprintf("--leader-elect-resource-namespace=%s", cluster.Namespace),
			fmt.Sprintf("--lock-object-namespace=%s", cluster.Namespace),
		},
	)

	deploymentTransforms = append(deploymentTransforms, argsTF)

	return declarative.NewBuilder(schedulerPackage, fs,
		declarative.WithManifestTransform(transform.ManifestTransform{
			"scheduler/deployment.yaml": deploymentTransforms,
		}),
		declarative.WithKustomizeMutationFunc([]kustomize.MutateFunc{
			kustomize.AddNamespace(cluster.GetNamespace()),
			kustomize.AddImages(images),
		}),
	)
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
