package storageoscluster

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	api "github.com/storageos/go-api/v2"
	"github.com/stretchr/testify/assert"

	storageoscomv1 "github.com/storageos/operator/api/v1"
	"github.com/storageos/operator/internal/storageos"
	"github.com/storageos/operator/internal/storageos/mocks"
)

func TestConfigureControlPlane(t *testing.T) {
	cases := []struct {
		name           string
		getClusterErr  error
		initialCluster *api.Cluster
		clusterSpec    storageoscomv1.StorageOSClusterSpec
		updatedCluster *api.UpdateClusterData
		wantErr        bool
	}{
		{
			name: "no update",
			initialCluster: &api.Cluster{
				DisableTelemetry:      false,
				DisableCrashReporting: false,
				DisableVersionCheck:   false,
				LogLevel:              "info",
				LogFormat:             "json",
			},
			clusterSpec: storageoscomv1.StorageOSClusterSpec{
				DisableTelemetry: false,
				Debug:            false,
			},
			updatedCluster: nil,
		},
		{
			name: "debug true",
			initialCluster: &api.Cluster{
				DisableTelemetry:      false,
				DisableCrashReporting: false,
				DisableVersionCheck:   false,
				LogLevel:              "info",
				LogFormat:             "json",
			},
			clusterSpec: storageoscomv1.StorageOSClusterSpec{
				DisableTelemetry: false,
				Debug:            true,
			},
			updatedCluster: &api.UpdateClusterData{
				DisableTelemetry:      false,
				DisableCrashReporting: false,
				DisableVersionCheck:   false,
				LogLevel:              "debug",
				LogFormat:             "json",
			},
		},
		{
			name: "telemetry disable",
			initialCluster: &api.Cluster{
				DisableTelemetry:      false,
				DisableCrashReporting: false,
				DisableVersionCheck:   false,
				LogLevel:              "info",
				LogFormat:             "json",
			},
			clusterSpec: storageoscomv1.StorageOSClusterSpec{
				DisableTelemetry: true,
				Debug:            false,
			},
			updatedCluster: &api.UpdateClusterData{
				DisableTelemetry:      true,
				DisableCrashReporting: true,
				DisableVersionCheck:   true,
				LogLevel:              "info",
				LogFormat:             "json",
			},
		},
		{
			name:          "api error",
			getClusterErr: errors.New("some api error"),
			wantErr:       true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Set up mocked control plane client.
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mcp := mocks.NewMockControlPlane(mockCtrl)
			stosCl := storageos.Mock(mcp)

			// Construct the cluster object.
			cluster := &storageoscomv1.StorageOSCluster{
				Spec: tc.clusterSpec,
			}

			initialCluster := api.Cluster{}
			if tc.initialCluster != nil {
				initialCluster = *tc.initialCluster
			}
			updatedCluster := api.UpdateClusterData{}
			if tc.updatedCluster != nil {
				updatedCluster = *tc.updatedCluster
			}

			// Set mock call expectations.
			mcp.EXPECT().GetCluster(gomock.Any()).Return(initialCluster, nil, tc.getClusterErr).Times(1)
			if tc.updatedCluster != nil {
				mcp.EXPECT().UpdateCluster(gomock.Any(), updatedCluster, gomock.Any()).Times(1)
			}

			err := configureControlPlane(context.TODO(), stosCl, cluster)
			if tc.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
