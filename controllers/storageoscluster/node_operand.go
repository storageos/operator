package storageoscluster

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/darkowlzz/operator-toolkit/declarative"
	"github.com/darkowlzz/operator-toolkit/declarative/kustomize"
	"github.com/darkowlzz/operator-toolkit/declarative/transform"
	eventv1 "github.com/darkowlzz/operator-toolkit/event/v1"
	"github.com/darkowlzz/operator-toolkit/operator/v1/operand"
	"go.opentelemetry.io/otel"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/filesys"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"

	storageoscomv1 "github.com/storageos/operator/api/v1"
	"github.com/storageos/operator/internal/image"
	stransform "github.com/storageos/operator/internal/transform"
)

const (
	// nodePackage contains the resource manifests for node operand.
	nodePackage = "node"

	// storageosContainer is the name of the storageos container.
	storageosContainer = "storageos"

	// Etcd TLS cert file names.
	tlsEtcdCA         = "etcd-client-ca.crt"
	tlsEtcdClientCert = "etcd-client.crt"
	tlsEtcdClientKey  = "etcd-client.key"

	// Etcd cert root path.
	tlsEtcdRootPath = "/run/storageos/pki"
)

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

	// Get image names.
	images := []kustomizetypes.Image{}
	namedImages := image.NamedImages{
		"storageos-init":            cluster.Spec.Images.InitContainer,
		"storageos-node":            cluster.Spec.Images.NodeContainer,
		"csi-node-driver-registrar": cluster.Spec.Images.CSINodeDriverRegistrarContainer,
		"csi-livenessprobe":         cluster.Spec.Images.CSILivenessProbeContainer,
	}
	images = append(images, image.GetKustomizeImageList(namedImages)...)

	// Create daemonset transforms.
	daemonsetTransforms := []transform.TransformFunc{}
	usernameTF, err := stransform.SetDaemonSetEnvVarValueFromSecretFunc(storageosContainer, "BOOTSTRAP_USERNAME", cluster.Spec.SecretRefName, "username")
	if err != nil {
		return nil, err
	}
	passwordTF, err := stransform.SetDaemonSetEnvVarValueFromSecretFunc(storageosContainer, "BOOTSTRAP_PASSWORD", cluster.Spec.SecretRefName, "password")
	if err != nil {
		return nil, err
	}
	daemonsetTransforms = append(daemonsetTransforms, usernameTF, passwordTF)

	// Create configmap transforms.
	configmapTransforms := []transform.TransformFunc{
		stransform.SetConfigMapData("ETCD_ENDPOINTS", cluster.Spec.KVBackend.Address),
		stransform.SetConfigMapData("DISABLE_TELEMETRY", strconv.FormatBool(cluster.Spec.DisableTelemetry)),
		// TODO: separte CR items for version check and crash reports.  Use
		// Telemetry to enable/disable everything for now.
		stransform.SetConfigMapData("DISABLE_VERSION_CHECK", strconv.FormatBool(cluster.Spec.DisableTelemetry)),
		stransform.SetConfigMapData("DISABLE_CRASH_REPORTING", strconv.FormatBool(cluster.Spec.DisableTelemetry)),
		stransform.SetConfigMapData("CSI_ENDPOINT", cluster.GetCSIEndpoint()),
		stransform.SetConfigMapData("LOG_LEVEL", cluster.GetLogLevel()),
	}

	// If etcd TLS related values are set, set the etcd related configurations.
	if cluster.Spec.TLSEtcdSecretRefName != "" && cluster.Spec.TLSEtcdSecretRefNamespace != "" {
		etcdConfig := []transform.TransformFunc{
			stransform.SetConfigMapData("ETCD_TLS_CLIENT_CA", filepath.Join(tlsEtcdRootPath, tlsEtcdCA)),
			stransform.SetConfigMapData("ETCD_TLS_CLIENT_KEY", filepath.Join(tlsEtcdRootPath, tlsEtcdClientKey)),
			stransform.SetConfigMapData("ETCD_TLS_CLIENT_CERT", filepath.Join(tlsEtcdRootPath, tlsEtcdClientCert)),
		}
		configmapTransforms = append(configmapTransforms, etcdConfig...)
	}

	if cluster.Spec.K8sDistro != "" {
		configmapTransforms = append(configmapTransforms, stransform.SetConfigMapData("K8S_DISTRO", cluster.Spec.K8sDistro))
	}

	if cluster.Spec.SharedDir != "" {
		configmapTransforms = append(configmapTransforms, stransform.SetConfigMapData("DEVICE_DIR", cluster.GetSharedDir()))
	}

	return declarative.NewBuilder(nodePackage, fs,
		declarative.WithManifestTransform(transform.ManifestTransform{
			"node/daemonset.yaml": daemonsetTransforms,
			"node/configmap.yaml": configmapTransforms,
		}),
		declarative.WithKustomizeMutationFunc([]kustomize.MutateFunc{
			kustomize.AddNamespace(cluster.GetNamespace()),
			kustomize.AddImages(images),
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
