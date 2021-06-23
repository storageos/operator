package storageoscluster

import (
	"context"
	"fmt"
	"os"

	"github.com/darkowlzz/operator-toolkit/declarative"
	"github.com/darkowlzz/operator-toolkit/declarative/kustomize"
	eventv1 "github.com/darkowlzz/operator-toolkit/event/v1"
	"github.com/darkowlzz/operator-toolkit/operator/v1/operand"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/filesys"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"

	storageoscomv1 "github.com/storageos/operator/api/v1"
	"github.com/storageos/operator/internal/image"
)

const (
	// csiPackage contains the resource manifests for csi operand.
	csiPackage = "csi"

	// Kustomize image name for container image.
	kImageCSIProvisioner = "csi-provisioner"
	kImageCSIAttacher    = "csi-attacher"
	kImageCSIResizer     = "csi-resizer"

	// Related image environment variable.
	csiProvisionerEnvVar = "RELATED_IMAGE_CSIV1_EXTERNAL_PROVISIONER"
	// TODO: Attacher env var has "V3" suffix for backwards compatibility.
	// Remove the suffix when doing a breaking change.
	csiAttacherEnvVar = "RELATED_IMAGE_CSIV1_EXTERNAL_ATTACHER_V3"
	csiResizerEnvVar  = "RELATED_IMAGE_CSIV1_EXTERNAL_RESIZER"
)

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
	ctx, span, _, log := instrumentation.Start(ctx, "CSIOperand.ReadyCheck")
	defer span.End()

	// Get the deployment object and check status of the replicas.
	csiDep := &appsv1.Deployment{}
	key := client.ObjectKey{Name: "storageos-csi-helper", Namespace: obj.GetNamespace()}
	if err := c.client.Get(ctx, key, csiDep); err != nil {
		return false, err
	}

	if csiDep.Status.AvailableReplicas > 0 {
		log.V(4).Info("Found available replicas more than 0", "availableReplicas", csiDep.Status.AvailableReplicas)
		return true, nil
	}

	log.V(4).Info("csi-helper not ready")
	return false, nil
}

func (c *CSIOperand) Ensure(ctx context.Context, obj client.Object, ownerRef metav1.OwnerReference) (eventv1.ReconcilerEvent, error) {
	ctx, span, _, _ := instrumentation.Start(ctx, "CSIOperand.Ensure")
	defer span.End()

	b, err := getCSIBuilder(c.fs, obj)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return nil, b.Apply(ctx)
}

func (c *CSIOperand) Delete(ctx context.Context, obj client.Object) (eventv1.ReconcilerEvent, error) {
	ctx, span, _, _ := instrumentation.Start(ctx, "CSIOperand.Delete")
	defer span.End()

	b, err := getCSIBuilder(c.fs, obj)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return nil, b.Delete(ctx)
}

func getCSIBuilder(fs filesys.FileSystem, obj client.Object) (*declarative.Builder, error) {
	cluster, ok := obj.(*storageoscomv1.StorageOSCluster)
	if !ok {
		return nil, fmt.Errorf("failed to convert %v to StorageOSCluster", obj)
	}

	// Get image names.
	images := []kustomizetypes.Image{}

	// Check environment variables for related images.
	relatedImages := image.NamedImages{
		kImageCSIProvisioner: os.Getenv(csiProvisionerEnvVar),
		kImageCSIAttacher:    os.Getenv(csiAttacherEnvVar),
		kImageCSIResizer:     os.Getenv(csiResizerEnvVar),
	}
	images = append(images, image.GetKustomizeImageList(relatedImages)...)

	// Get the images from the cluster spec. These overwrite the default images
	// set by the operator related images environment variables.
	namedImages := image.NamedImages{
		kImageCSIProvisioner: cluster.Spec.Images.CSIExternalProvisionerContainer,
		kImageCSIAttacher:    cluster.Spec.Images.CSIExternalAttacherContainer,
		kImageCSIResizer:     cluster.Spec.Images.CSIExternalResizerContainer,
	}
	images = append(images, image.GetKustomizeImageList(namedImages)...)

	return declarative.NewBuilder(csiPackage, fs,
		declarative.WithKustomizeMutationFunc([]kustomize.MutateFunc{
			kustomize.AddNamespace(cluster.GetNamespace()),
			kustomize.AddImages(images),
		}),
	)
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
