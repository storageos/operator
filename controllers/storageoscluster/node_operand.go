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
	corev1 "k8s.io/api/core/v1"
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

	// storageosService is the default name of the storageos service.
	storageosService = "storageos"

	// Etcd TLS cert file names.
	tlsEtcdCA         = "etcd-client-ca.crt"
	tlsEtcdClientCert = "etcd-client.crt"
	tlsEtcdClientKey  = "etcd-client.key"

	// Etcd cert root path.
	tlsEtcdRootPath = "/run/storageos/pki"

	// Etcd certs volume name.
	tlsEtcdCertsVolume = "etcd-certs"

	// Shared device directory volume name.
	sharedDirVolume = "shared"
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

	// TODO: Apply only at creation. Subsequent updates should only update
	// individual properties.
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

	// Create transforms for setting bootstrap credentials.
	usernameTF := stransform.SetDaemonSetEnvVarValueFromSecretFunc(storageosContainer, "BOOTSTRAP_USERNAME", cluster.Spec.SecretRefName, "username")
	passwordTF := stransform.SetDaemonSetEnvVarValueFromSecretFunc(storageosContainer, "BOOTSTRAP_PASSWORD", cluster.Spec.SecretRefName, "password")
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

	// If etcd TLS related values are set, mount the secret volume and set the
	// etcd related configurations.
	if cluster.Spec.TLSEtcdSecretRefName != "" {
		// Add etcd secret volume transform.
		etcdSecretVolTF := stransform.SetDaemonSetSecretVolumeFunc("etcd-certs", cluster.Spec.TLSEtcdSecretRefName, nil)

		// Add etcd secret volume mount transform.
		etcdSecretVolMountTF := stransform.SetDaemonSetVolumeMountFunc(storageosContainer, tlsEtcdCertsVolume, tlsEtcdRootPath, "")
		daemonsetTransforms = append(daemonsetTransforms, etcdSecretVolTF, etcdSecretVolMountTF)

		// Add etcd secret configuration transforms.
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

	// If shared dir is set, mount the device as host path volume and set the
	// configuration.
	if cluster.Spec.SharedDir != "" {
		// Add shared device volume transform.
		sharedDeviceVolTF := stransform.SetDaemonSetHostPathVolumeFunc(sharedDirVolume, cluster.Spec.SharedDir, nil)

		// Add shared device volumemount transform.
		sharedDeviceVolMountTF := stransform.SetDaemonSetVolumeMountFunc(storageosContainer, sharedDirVolume, cluster.Spec.SharedDir, "")
		daemonsetTransforms = append(daemonsetTransforms, sharedDeviceVolTF, sharedDeviceVolMountTF)

		// Add shared device configuration transform.
		configmapTransforms = append(configmapTransforms, stransform.SetConfigMapData("DEVICE_DIR", cluster.GetSharedDir()))
	}

	// If node selector terms are provided, append the node selectors.
	if len(cluster.Spec.NodeSelectorTerms) > 0 {
		daemonsetTransforms = append(daemonsetTransforms, stransform.SetDaemonSetNodeSelectorTermsFunc(cluster.Spec.NodeSelectorTerms))
	}

	// Add the default tolerations.
	tolerations := getDefaultTolerations()
	// If tolerations are provided, append the tolerations.
	if cluster.Spec.Tolerations != nil {
		tolerations = append(tolerations, cluster.Spec.Tolerations...)
	}
	daemonsetTransforms = append(daemonsetTransforms, stransform.SetDaemonSetTolerationFunc(tolerations))

	// If any resources are defined, set container resource requirements.
	if cluster.Spec.Resources.Limits != nil || cluster.Spec.Resources.Requests != nil {
		daemonsetTransforms = append(daemonsetTransforms, stransform.SetDaemonSetContainerResourceFunc(storageosContainer, cluster.Spec.Resources))
	}

	// Create service transforms.
	serviceTransforms := []transform.TransformFunc{}
	serviceName := storageosService

	// If a service name is provided, set the service names.
	if cluster.Spec.Service.Name != "" {
		serviceName = cluster.Spec.Service.Name
		serviceTransforms = append(
			serviceTransforms,
			stransform.SetServiceNameFunc(serviceName),
			stransform.SetDefaultServicePortNameFunc(serviceName),
		)
	}
	if cluster.Spec.Service.Type != "" {
		serviceTransforms = append(serviceTransforms, stransform.SetServiceTypeFunc(corev1.ServiceType(cluster.Spec.Service.Type)))
	}
	if cluster.Spec.Service.InternalPort != 0 {
		serviceTransforms = append(serviceTransforms, stransform.SetServiceInternalPortFunc(serviceName, cluster.Spec.Service.InternalPort))
	}
	if cluster.Spec.Service.ExternalPort != 0 {
		serviceTransforms = append(serviceTransforms, stransform.SetServiceExternalPortFunc(serviceName, cluster.Spec.Service.ExternalPort))
	}
	if len(cluster.Spec.Service.Annotations) > 0 {
		serviceTransforms = append(serviceTransforms, transform.AddAnnotationsFunc(cluster.Spec.Service.Annotations))
	}

	return declarative.NewBuilder(nodePackage, fs,
		declarative.WithManifestTransform(transform.ManifestTransform{
			"node/daemonset.yaml": daemonsetTransforms,
			"node/configmap.yaml": configmapTransforms,
			"node/service.yaml":   serviceTransforms,
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
