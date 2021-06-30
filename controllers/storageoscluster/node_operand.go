package storageoscluster

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/darkowlzz/operator-toolkit/declarative"
	"github.com/darkowlzz/operator-toolkit/declarative/kustomize"
	"github.com/darkowlzz/operator-toolkit/declarative/transform"
	eventv1 "github.com/darkowlzz/operator-toolkit/event/v1"
	"github.com/darkowlzz/operator-toolkit/operator/v1/operand"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/filesys"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"

	storageoscomv1 "github.com/storageos/operator/api/v1"
	"github.com/storageos/operator/internal/image"
	"github.com/storageos/operator/internal/storageos"
	stransform "github.com/storageos/operator/internal/transform"
)

const (
	// nodePackage contains the resource manifests for node operand.
	nodePackage = "node"

	// storageosContainer is the name of the storageos container.
	storageosContainer = "storageos"

	// initContainer is the name of the storageos init container.
	initContainer = "storageos-init"

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

	// Kustomize image names for all the container images.
	kImageInit             = "storageos-init"
	kImageNode             = "storageos-node"
	kImageCSINodeDriverReg = "csi-node-driver-registrar"
	kImageCSILivenessProbe = "csi-livenessprobe"

	// Related image environment variables.
	initImageEnvVar             = "RELATED_IMAGE_STORAGEOS_INIT"
	nodeImageEnvVar             = "RELATED_IMAGE_STORAGEOS_NODE"
	csiNodeDriverRegImageEnvVar = "RELATED_IMAGE_CSIV1_NODE_DRIVER_REGISTRAR"
	csiLivenessProbeImageEnvVar = "RELATED_IMAGE_CSIV1_LIVENESS_PROBE"
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
	ctx, span, _, log := instrumentation.Start(ctx, "NodeOperand.ReadyCheck")
	defer span.End()

	// Get the DaemonSet object and check the status of the ready instances.
	// One ready instance should be enough for the installation to continue.
	// Other components that depend on control-plane should be able to connect
	// to it.
	nodeDS := &appsv1.DaemonSet{}
	key := client.ObjectKey{Name: "storageos-daemonset", Namespace: obj.GetNamespace()}
	if err := c.client.Get(ctx, key, nodeDS); err != nil {
		return false, err
	}

	if nodeDS.Status.NumberReady > 0 {
		log.V(4).Info("Found more than 0 ready nodes", "NumberReady", nodeDS.Status.NumberReady)
		return true, nil
	}

	log.V(4).Info("storageos-daemonset not ready")
	return false, nil
}

func (c *NodeOperand) Ensure(ctx context.Context, obj client.Object, ownerRef metav1.OwnerReference) (eventv1.ReconcilerEvent, error) {
	ctx, span, _, _ := instrumentation.Start(ctx, "NodeOperand.Ensure")
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
	ctx, span, _, _ := instrumentation.Start(ctx, "NodeOperand.Delete")
	defer span.End()

	b, err := getNodeBuilder(c.fs, obj)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return nil, b.Delete(ctx)
}

// PostReady performs actions after the control-plane is ready.
func (c *NodeOperand) PostReady(ctx context.Context, obj client.Object) error {
	ctx, span, _, _ := instrumentation.Start(ctx, "NodeOperand.PostReady")
	defer span.End()

	cluster, ok := obj.(*storageoscomv1.StorageOSCluster)
	if !ok {
		return fmt.Errorf("failed to convert %v to StorageOSCluster", obj)
	}

	// Get a control plane client and configure the cluster.
	stosCl, err := getControlPlaneClient(ctx, c.client, cluster)
	if err != nil {
		return err
	}
	return configureControlPlane(ctx, stosCl, cluster)
}

func getNodeBuilder(fs filesys.FileSystem, obj client.Object) (*declarative.Builder, error) {
	cluster, ok := obj.(*storageoscomv1.StorageOSCluster)
	if !ok {
		return nil, fmt.Errorf("failed to convert %v to StorageOSCluster", obj)
	}

	// Get image names.
	images := []kustomizetypes.Image{}

	// Check environment variables for related images.
	relatedImages := image.NamedImages{
		kImageInit:             os.Getenv(initImageEnvVar),
		kImageNode:             os.Getenv(nodeImageEnvVar),
		kImageCSINodeDriverReg: os.Getenv(csiNodeDriverRegImageEnvVar),
		kImageCSILivenessProbe: os.Getenv(csiLivenessProbeImageEnvVar),
	}
	images = append(images, image.GetKustomizeImageList(relatedImages)...)

	// Get the images from the cluster spec. These overwrite the default images
	// set by the operator related images environment variables.
	namedImages := image.NamedImages{
		kImageInit:             cluster.Spec.Images.InitContainer,
		kImageNode:             cluster.Spec.Images.NodeContainer,
		kImageCSINodeDriverReg: cluster.Spec.Images.CSINodeDriverRegistrarContainer,
		kImageCSILivenessProbe: cluster.Spec.Images.CSILivenessProbeContainer,
	}
	images = append(images, image.GetKustomizeImageList(namedImages)...)

	// Create daemonset transforms.
	daemonsetTransforms := []transform.TransformFunc{}

	// Create transforms for setting bootstrap credentials.
	usernameTF := stransform.SetPodTemplateContainerEnvVarValueFromSecretFunc(storageosContainer, "BOOTSTRAP_USERNAME", cluster.Spec.SecretRefName, "username")
	passwordTF := stransform.SetPodTemplateContainerEnvVarValueFromSecretFunc(storageosContainer, "BOOTSTRAP_PASSWORD", cluster.Spec.SecretRefName, "password")

	// Set the init container env var.
	initNamespaceTF := stransform.SetPodTemplateInitContainerEnvVarStringFunc(initContainer, "DAEMONSET_NAMESPACE", cluster.GetNamespace())

	daemonsetTransforms = append(daemonsetTransforms, usernameTF, passwordTF, initNamespaceTF)

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
		etcdSecretVolTF := stransform.SetPodTemplateSecretVolumeFunc("etcd-certs", cluster.Spec.TLSEtcdSecretRefName, nil)

		// Add etcd secret volume mount transform.
		etcdSecretVolMountTF := stransform.SetPodTemplateVolumeMountFunc(storageosContainer, tlsEtcdCertsVolume, tlsEtcdRootPath, "")
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
		sharedDeviceVolTF := stransform.SetPodTemplateHostPathVolumeFunc(sharedDirVolume, cluster.Spec.SharedDir, nil)

		// Add shared device volumemount transform.
		sharedDeviceVolMountTF := stransform.SetPodTemplateVolumeMountFunc(storageosContainer, sharedDirVolume, cluster.Spec.SharedDir, "")
		daemonsetTransforms = append(daemonsetTransforms, sharedDeviceVolTF, sharedDeviceVolMountTF)

		// Add shared device configuration transform.
		configmapTransforms = append(configmapTransforms, stransform.SetConfigMapData("DEVICE_DIR", cluster.GetSharedDir()))
	}

	// If node selector terms are provided, append the node selectors.
	if len(cluster.Spec.NodeSelectorTerms) > 0 {
		daemonsetTransforms = append(daemonsetTransforms, stransform.SetPodTemplateNodeSelectorTermsFunc(cluster.Spec.NodeSelectorTerms))
	}

	// Add the default tolerations.
	tolerations := getDefaultTolerations()
	// If tolerations are provided, append the tolerations.
	if cluster.Spec.Tolerations != nil {
		tolerations = append(tolerations, cluster.Spec.Tolerations...)
	}
	daemonsetTransforms = append(daemonsetTransforms, stransform.SetPodTemplateTolerationFunc(tolerations))

	// If any resources are defined, set container resource requirements.
	if cluster.Spec.Resources.Limits != nil || cluster.Spec.Resources.Requests != nil {
		daemonsetTransforms = append(daemonsetTransforms, stransform.SetPodTemplateContainerResourceFunc(storageosContainer, cluster.Spec.Resources))
	}

	// Create service transforms.
	serviceTransforms := []transform.TransformFunc{}
	serviceName := storageosService

	// If a service name is provided, set the service names.
	if cluster.Spec.Service.Name != "" {
		serviceName = cluster.Spec.Service.Name
		serviceTransforms = append(
			serviceTransforms,
			stransform.SetMetadataNameFunc(serviceName),
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

// getControlPlaneClient creates an authenticated control plane client and
// returns it.
func getControlPlaneClient(ctx context.Context, kcl client.Client, cluster *storageoscomv1.StorageOSCluster) (*storageos.Client, error) {
	ctx, span, _, _ := instrumentation.Start(ctx, "getControlPlaneClient")
	defer span.End()

	// Get storageos creds and configure a client.
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{Name: cluster.Spec.SecretRefName, Namespace: cluster.GetNamespace()}
	if err := kcl.Get(ctx, secretKey, secret); err != nil {
		return nil, fmt.Errorf("failed to get storageos credentials: %w", err)
	}

	cpEndpoint := fmt.Sprintf("%s://%s.%s.svc:%d",
		storageos.DefaultScheme, storageosService,
		cluster.GetNamespace(), storageos.DefaultPort,
	)
	stosCl, err := storageos.New(cpEndpoint)
	if err != nil {
		return nil, err
	}

	if err := stosCl.Authenticate(ctx,
		string(secret.Data[storageos.UsernameKey]),
		string(secret.Data[storageos.PasswordKey])); err != nil {
		return nil, err
	}
	return stosCl, nil
}

// configureControlPlane takes the desired cluster configuration and
// reconfigures the control-plane.
func configureControlPlane(ctx context.Context, stosCl *storageos.Client, cluster *storageoscomv1.StorageOSCluster) error {
	ctx, span, _, log := instrumentation.Start(ctx, "configureControlPlane")
	defer span.End()

	// Get current cluster config.
	currentConfig, err := stosCl.GetCluster(ctx)
	if err != nil {
		return err
	}

	// Construct desired configuration.
	desiredConfig := &storageos.Cluster{
		DisableTelemetry:      cluster.Spec.DisableTelemetry,
		DisableCrashReporting: cluster.Spec.DisableTelemetry,
		DisableVersionCheck:   cluster.Spec.DisableTelemetry,
		LogLevel:              currentConfig.LogLevel,
		LogFormat:             currentConfig.LogFormat,
		Version:               currentConfig.Version,
	}
	if cluster.Spec.Debug {
		desiredConfig.LogLevel = storageos.LogLevelDebug
	}

	// Compare the current and desired configuration and update if necessary.
	if currentConfig.IsEqual(desiredConfig) {
		return nil
	}
	log.Info("current config doesn't match the desired config, applying update")
	return stosCl.UpdateCluster(ctx, desiredConfig)
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
