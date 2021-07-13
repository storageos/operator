package v1

import (
	"fmt"
)

const (
	// defaultPluginRegistrationPath is the default kubelet plugin registration
	// directory path.
	defaultPluginRegistrationPath = "/var/lib/kubelet/plugins_registry"

	// defaultCSIEndpoint is the default path to the storageos CSI socket.
	defaultCSIEndpoint = "/storageos/csi.sock"

	// Log levels.
	debugLogLevel = "debug"
	infoLogLevel  = "info"
)

// GetCSIEndpoint returns the CSI endpoint for the cluster.
func (s *StorageOSCluster) GetCSIEndpoint() string {
	if s.Spec.CSI.Endpoint != "" {
		return s.Spec.CSI.Endpoint
	}
	return fmt.Sprintf("%s%s%s", "unix://", defaultPluginRegistrationPath, defaultCSIEndpoint)
}

// GetSharedDir returns the shared directory of the cluster.
func (s *StorageOSCluster) GetSharedDir() string {
	if s.Spec.SharedDir != "" {
		return s.Spec.SharedDir
	}
	return fmt.Sprintf("%s/devices", s.Spec.SharedDir)
}

// GetLogLevel returns the log level of the cluster.
func (s *StorageOSCluster) GetLogLevel() string {
	if s.Spec.Debug {
		return debugLogLevel
	}
	return infoLogLevel
}
