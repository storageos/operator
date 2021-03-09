package storageoscluster

import (
	"context"
	"errors"
	"fmt"

	"github.com/darkowlzz/operator-toolkit/declarative"
	"github.com/darkowlzz/operator-toolkit/declarative/transform"
	eventv1 "github.com/darkowlzz/operator-toolkit/event/v1"
	"github.com/darkowlzz/operator-toolkit/operator/v1/operand"
	"go.opentelemetry.io/otel"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/filesys"

	storageoscomv1 "github.com/storageos/operator/api/v1"
	stransform "github.com/storageos/operator/internal/transform"
)

const (
	// storageclassPackage contains the resource manifests for storageclass
	// operand.
	storageclassPackage = "storageclass"

	// csiSecretNameKey is the StorageClass parameter key for CSI secret name.
	csiSecretNameKey = "csi.storage.k8s.io/secret-name"

	// csiSecretNamespaceKey is the StorageClass parameter key for CSI secret
	// namespace.
	csiSecretNamespaceKey = "csi.storage.k8s.io/secret-namespace"

	// storageClassParameterPath is the path to StorageClass parameters.
	storageClassParametersPath = "parameters"
)

type StorageClassOperand struct {
	name            string
	client          client.Client
	requires        []string
	requeueStrategy operand.RequeueStrategy
	fs              filesys.FileSystem
}

var _ operand.Operand = &StorageClassOperand{}

func (sc *StorageClassOperand) Name() string                             { return sc.name }
func (sc *StorageClassOperand) Requires() []string                       { return sc.requires }
func (sc *StorageClassOperand) RequeueStrategy() operand.RequeueStrategy { return sc.requeueStrategy }
func (sc *StorageClassOperand) ReadyCheck(ctx context.Context, obj client.Object) (bool, error) {
	return true, nil
}

func (sc *StorageClassOperand) Ensure(ctx context.Context, obj client.Object, ownerRef metav1.OwnerReference) (eventv1.ReconcilerEvent, error) {
	// Setup a tracer and start a span.
	tr := otel.Tracer("StorageClassOperand Ensure")
	ctx, span := tr.Start(ctx, "storageclassoperand ensure")
	defer span.End()

	b, err := getStorageClassBuilder(sc.fs, obj)
	if err != nil {
		if errors.Is(err, noResourceErr) {
			log.Info("no storageclass specified")
			return nil, nil
		}
		span.RecordError(err)
		return nil, err
	}

	return nil, b.Apply(ctx)
}

func (sc *StorageClassOperand) Delete(ctx context.Context, obj client.Object) (eventv1.ReconcilerEvent, error) {
	// Setup a tracer and start a span.
	tr := otel.Tracer("StorageClassOperand Delete")
	ctx, span := tr.Start(ctx, "storageclassoperand delete")
	defer span.End()

	b, err := getStorageClassBuilder(sc.fs, obj)
	if err != nil {
		if errors.Is(err, noResourceErr) {
			return nil, nil
		}
		span.RecordError(err)
		return nil, err
	}

	return nil, b.Delete(ctx)
}

func getStorageClassBuilder(fs filesys.FileSystem, obj client.Object) (*declarative.Builder, error) {
	cluster, ok := obj.(*storageoscomv1.StorageOSCluster)
	if !ok {
		return nil, fmt.Errorf("failed to convert %v to StorageOSCluster", obj)
	}

	// Skip if no StorageClass name is provided.
	if cluster.Spec.StorageClassName == "" {
		return nil, noResourceErr
	}

	// StorageClass transforms.
	scTransforms := []transform.TransformFunc{}

	// Set the StorageClass name.
	nameTF := stransform.SetMetadataNameFunc(cluster.Spec.StorageClassName)

	// Set secret reference.
	secretNameTF := stransform.SetScalarNodeStringValueFunc(csiSecretNameKey, cluster.Spec.SecretRefName, storageClassParametersPath)
	secretNamespaceTF := stransform.SetScalarNodeStringValueFunc(csiSecretNamespaceKey, cluster.Namespace, storageClassParametersPath)

	scTransforms = append(scTransforms, nameTF, secretNameTF, secretNamespaceTF)

	return declarative.NewBuilder(storageclassPackage, fs,
		declarative.WithManifestTransform(transform.ManifestTransform{
			"storageclass/storageclass.yaml": scTransforms,
		}),
	)
}

func NewStorageClassOperand(
	name string,
	client client.Client,
	requires []string,
	requeueStrategy operand.RequeueStrategy,
	fs filesys.FileSystem,
) *StorageClassOperand {
	return &StorageClassOperand{
		name:            name,
		client:          client,
		requires:        requires,
		requeueStrategy: requeueStrategy,
		fs:              fs,
	}
}
