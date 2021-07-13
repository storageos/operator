package storageoscluster

import (
	"context"
	"fmt"
	"os"

	"github.com/darkowlzz/operator-toolkit/declarative"
	"github.com/darkowlzz/operator-toolkit/declarative/kustomize"
	"github.com/darkowlzz/operator-toolkit/declarative/transform"
	eventv1 "github.com/darkowlzz/operator-toolkit/event/v1"
	"github.com/darkowlzz/operator-toolkit/operator/v1/operand"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/filesys"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"

	storageoscomv1 "github.com/storageos/operator/apis/v1"
	"github.com/storageos/operator/internal/image"
	stransform "github.com/storageos/operator/internal/transform"
)

const (
	// schedulerPackage contains the resource manifests for scheduler operand.
	schedulerPackage = "scheduler"

	// Kustomize image name for container image.
	kImageKubeScheduler = "kube-scheduler"

	// Related image environment variable.
	kubeSchedulerEnvVar = "RELATED_IMAGE_KUBE_SCHEDULER"
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
	ctx, span, _, log := instrumentation.Start(ctx, "SchedulerOperand.ReadyCheck")
	defer span.End()

	// Get the deployment object and check status of the replicas.
	schedulerDep := &appsv1.Deployment{}
	key := client.ObjectKey{Name: "storageos-scheduler", Namespace: obj.GetNamespace()}
	if err := c.client.Get(ctx, key, schedulerDep); err != nil {
		return false, err
	}

	if schedulerDep.Status.AvailableReplicas > 0 {
		log.V(4).Info("Found available replicas more than 0", "availableReplicas", schedulerDep.Status.AvailableReplicas)
		return true, nil
	}

	log.V(4).Info("scheduler not ready")
	return false, nil
}

func (c *SchedulerOperand) PostReady(ctx context.Context, obj client.Object) error { return nil }

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

	// Check environment variables for related images.
	relatedImages := image.NamedImages{
		kImageKubeScheduler: os.Getenv(kubeSchedulerEnvVar),
	}
	images = append(images, image.GetKustomizeImageList(relatedImages)...)

	// Get the images from the cluster spec. This overwrites the default image
	// set by the operator related images environment variables.
	namedImages := image.NamedImages{
		kImageKubeScheduler: cluster.Spec.Images.KubeSchedulerContainer,
	}
	images = append(images, image.GetKustomizeImageList(namedImages)...)

	// Create kubescheduler config transforms.
	configTransforms := []transform.TransformFunc{}

	// Add leader election resource lock namespace.
	rnsTF := stransform.SetKubeSchedulerLeaderElectionRNamespaceFunc(cluster.Namespace)

	configTransforms = append(configTransforms, rnsTF)

	return declarative.NewBuilder(schedulerPackage, fs,
		declarative.WithManifestTransform(transform.ManifestTransform{
			"scheduler/config.yaml": configTransforms,
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
