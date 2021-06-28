package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// StorageOSClusterSpec defines the desired state of StorageOSCluster
type StorageOSClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Join is the join token used for service discovery.
	Join string `json:"join,omitempty"`

	// CSI defines the configurations for CSI.
	CSI StorageOSClusterCSI `json:"csi,omitempty"`

	// Namespace is the kubernetes Namespace where storageos resources are
	// provisioned.
	Namespace string `json:"namespace,omitempty"`

	// StorageClassName is the name of default StorageClass created for
	// StorageOS volumes.
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	StorageClassName string `json:"storageClassName,omitempty"`

	// Service is the Service configuration for the cluster nodes.
	Service StorageOSClusterService `json:"service,omitempty"`

	// SecretRefName is the name of the secret object that contains all the
	// sensitive cluster configurations.
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	SecretRefName string `json:"secretRefName"`

	// SecretRefNamespace is the namespace of the secret reference.
	SecretRefNamespace string `json:"secretRefNamespace,omitempty"`

	// SharedDir is the shared directory to be used when the kubelet is running
	// in a container.
	// Typically: "/var/lib/kubelet/plugins/kubernetes.io~storageos".
	// If not set, defaults will be used.
	SharedDir string `json:"sharedDir,omitempty"`

	// Ingress defines the ingress configurations used in the cluster.
	Ingress StorageOSClusterIngress `json:"ingress,omitempty"`

	// Images defines the various container images used in the cluster.
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	Images ContainerImages `json:"images,omitempty"`

	// KVBackend defines the key-value store backend used in the cluster.
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	KVBackend StorageOSClusterKVBackend `json:"kvBackend"`

	// Pause is to pause the operator for the cluster.
	Pause bool `json:"pause,omitempty"`

	// Debug is to set debug mode of the cluster.
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	Debug bool `json:"debug,omitempty"`

	// NodeSelectorTerms is to set the placement of storageos pods using
	// node affinity requiredDuringSchedulingIgnoredDuringExecution.
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	NodeSelectorTerms []corev1.NodeSelectorTerm `json:"nodeSelectorTerms,omitempty"`

	// Tolerations is to set the placement of storageos pods using
	// pod toleration.
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Resources is to set the resource requirements of the storageos containers.
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Disable Pod Fencing.  With StatefulSets, Pods are only re-scheduled if
	// the Pod has been marked as killed.  In practice this means that failover
	// of a StatefulSet pod is a manual operation.
	//
	// By enabling Pod Fencing and setting the `storageos.com/fenced=true` label
	// on a Pod, StorageOS will enable automated Pod failover (by killing the
	// application Pod on the failed node) if the following conditions exist:
	//
	// - Pod fencing has not been explicitly disabled.
	// - StorageOS has determined that the node the Pod is running on is
	//   offline.  StorageOS uses Gossip and TCP checks and will retry for 30
	//   seconds.  At this point all volumes on the failed node are marked
	//   offline (irrespective of whether fencing is enabled) and volume
	//   failover starts.
	// - The Pod has the label `storageos.com/fenced=true` set.
	// - The Pod has at least one StorageOS volume attached.
	// - Each StorageOS volume has at least 1 healthy replica.
	//
	// When Pod Fencing is disabled, StorageOS will not perform any interaction
	// with Kubernetes when it detects that a node has gone offline.
	// Additionally, the Kubernetes permissions required for Fencing will not be
	// added to the StorageOS role.
	DisableFencing bool `json:"disableFencing,omitempty"`

	// Disable Telemetry.
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	DisableTelemetry bool `json:"disableTelemetry,omitempty"`

	// Disable TCMU can be set to true to disable the TCMU storage driver.  This
	// is required when there are multiple storage systems running on the same
	// node and you wish to avoid conflicts.  Only one TCMU-based storage system
	// can run on a node at a time.
	//
	// Disabling TCMU will degrade performance.
	DisableTCMU bool `json:"disableTCMU,omitempty"`

	// Force TCMU can be set to true to ensure that TCMU is enabled or
	// cause StorageOS to abort startup.
	//
	// At startup, StorageOS will automatically fallback to non-TCMU mode if
	// another TCMU-based storage system is running on the node.  Since non-TCMU
	// will degrade performance, this may not always be desired.
	ForceTCMU bool `json:"forceTCMU,omitempty"`

	// TLSEtcdSecretRefName is the name of the secret object that contains the
	// etcd TLS certs. This secret is shared with etcd, therefore it's not part
	// of the main storageos secret.
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	TLSEtcdSecretRefName string `json:"tlsEtcdSecretRefName,omitempty"`

	// TLSEtcdSecretRefNamespace is the namespace of the etcd TLS secret object.
	TLSEtcdSecretRefNamespace string `json:"tlsEtcdSecretRefNamespace,omitempty"`

	// K8sDistro is the name of the Kubernetes distribution where the operator
	// is being deployed.  It should be in the format: `name[-1.0]`, where the
	// version is optional and should only be appended if known.  Suitable names
	// include: `openshift`, `rancher`, `aks`, `gke`, `eks`, or the deployment
	// method if using upstream directly, e.g `minishift` or `kubeadm`.
	//
	// Setting k8sDistro is optional, and will be used to simplify cluster
	// configuration by setting appropriate defaults for the distribution.  The
	// distribution information will also be included in the product telemetry
	// (if enabled), to help focus development efforts.
	K8sDistro string `json:"k8sDistro,omitempty"`

	// Disable StorageOS scheduler extender.
	DisableScheduler bool `json:"disableScheduler,omitempty"`
}

// ContainerImages contains image names of all the containers used by the operator.
type ContainerImages struct {
	NodeContainer                      string `json:"nodeContainer,omitempty"`
	InitContainer                      string `json:"initContainer,omitempty"`
	CSINodeDriverRegistrarContainer    string `json:"csiNodeDriverRegistrarContainer,omitempty"`
	CSIClusterDriverRegistrarContainer string `json:"csiClusterDriverRegistrarContainer,omitempty"`
	CSIExternalProvisionerContainer    string `json:"csiExternalProvisionerContainer,omitempty"`
	CSIExternalAttacherContainer       string `json:"csiExternalAttacherContainer,omitempty"`
	CSIExternalResizerContainer        string `json:"csiExternalResizerContainer,omitempty"`
	CSILivenessProbeContainer          string `json:"csiLivenessProbeContainer,omitempty"`
	HyperkubeContainer                 string `json:"hyperkubeContainer,omitempty"`
	KubeSchedulerContainer             string `json:"kubeSchedulerContainer,omitempty"`
	NFSContainer                       string `json:"nfsContainer,omitempty"`
	APIManagerContainer                string `json:"apiManagerContainer,omitempty"`
}

// StorageOSClusterCSI contains CSI configurations.
type StorageOSClusterCSI struct {
	Enable                       bool   `json:"enable,omitempty"`
	Version                      string `json:"version,omitempty"`
	Endpoint                     string `json:"endpoint,omitempty"`
	EnableProvisionCreds         bool   `json:"enableProvisionCreds,omitempty"`
	EnableControllerPublishCreds bool   `json:"enableControllerPublishCreds,omitempty"`
	EnableNodePublishCreds       bool   `json:"enableNodePublishCreds,omitempty"`
	EnableControllerExpandCreds  bool   `json:"enableControllerExpandCreds,omitempty"`
	RegistrarSocketDir           string `json:"registrarSocketDir,omitempty"`
	KubeletDir                   string `json:"kubeletDir,omitempty"`
	PluginDir                    string `json:"pluginDir,omitempty"`
	DeviceDir                    string `json:"deviceDir,omitempty"`
	RegistrationDir              string `json:"registrationDir,omitempty"`
	KubeletRegistrationPath      string `json:"kubeletRegistrationPath,omitempty"`
	DriverRegistrationMode       string `json:"driverRegisterationMode,omitempty"`
	DriverRequiresAttachment     string `json:"driverRequiresAttachment,omitempty"`
	DeploymentStrategy           string `json:"deploymentStrategy,omitempty"`
}

// StorageOSClusterService contains Service configurations.
type StorageOSClusterService struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	ExternalPort int               `json:"externalPort,omitempty"`
	InternalPort int               `json:"internalPort,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
}

// StorageOSClusterIngress contains Ingress configurations.
type StorageOSClusterIngress struct {
	Enable      bool              `json:"enable,omitempty"`
	Hostname    string            `json:"hostname,omitempty"`
	TLS         bool              `json:"tls,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// StorageOSClusterKVBackend stores key-value store backend configurations.
type StorageOSClusterKVBackend struct {
	Address string `json:"address"`
	Backend string `json:"backend,omitempty"`
}

// StorageOSClusterStatus defines the observed state of StorageOSCluster
type StorageOSClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Phase is the phase of the StorageOS cluster.
	//+operator-sdk:csv:customresourcedefinitions:type=status
	Phase string `json:"phase,omitempty"`

	NodeHealthStatus map[string]NodeHealth `json:"nodeHealthStatus,omitempty"`
	Nodes            []string              `json:"nodes,omitempty"`

	// Ready is the ready status of the StorageOS control-plane pods.
	//+operator-sdk:csv:customresourcedefinitions:type=status
	Ready string `json:"ready,omitempty"`

	// Members is the list of StorageOS nodes in the cluster.
	//+operator-sdk:csv:customresourcedefinitions:type=status
	Members MembersStatus `json:"members,omitempty"`
	// Conditions is a list of status of all the components of StorageOS.
	//+operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// NodeHealth contains health status of a node.
type NodeHealth struct {
	DirectfsInitiator string `json:"directfsInitiator,omitempty"`
	Director          string `json:"director,omitempty"`
	KV                string `json:"kv,omitempty"`
	KVWrite           string `json:"kvWrite,omitempty"`
	Nats              string `json:"nats,omitempty"`
	Presentation      string `json:"presentation,omitempty"`
	Rdb               string `json:"rdb,omitempty"`
}

// MembersStatus stores the status details of cluster member nodes.
type MembersStatus struct {
	// Ready are the storageos cluster members that are ready to serve requests.
	// The member names are the same as the node IPs.
	Ready []string `json:"ready,omitempty"`
	// Unready are the storageos cluster nodes not ready to serve requests.
	Unready []string `json:"unready,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ready",type="string",JSONPath=".status.ready",description="Ready status of the storageos nodes."
// +kubebuilder:printcolumn:name="status",type="string",JSONPath=".status.phase",description="Status of the whole cluster."
// +kubebuilder:printcolumn:name="age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:path=storageosclusters,shortName=stos

// StorageOSCluster is the Schema for the storageosclusters API
//+operator-sdk:csv:customresourcedefinitions:displayName="StorageOS Cluster",resources={{DaemonSet,apps/v1,storageos-daemonset},{Deployment,apps/v1,storageos-api-manager},{Deployment,apps/v1,storageos-csi-helper},{Deployment,apps/v1,storageos-scheduler}}
type StorageOSCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StorageOSClusterSpec   `json:"spec,omitempty"`
	Status StorageOSClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StorageOSClusterList contains a list of StorageOSCluster
type StorageOSClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StorageOSCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StorageOSCluster{}, &StorageOSClusterList{})
}
