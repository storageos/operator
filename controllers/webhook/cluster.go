package webhook

import (
	"context"
	"fmt"

	"github.com/darkowlzz/operator-toolkit/singleton"
	"github.com/darkowlzz/operator-toolkit/telemetry"
	tkadmission "github.com/darkowlzz/operator-toolkit/webhook/admission"
	"github.com/darkowlzz/operator-toolkit/webhook/builder"
	"github.com/darkowlzz/operator-toolkit/webhook/function"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	storageoscomv1 "github.com/storageos/operator/apis/v1"
)

const (
	// instrumentationName is the instrumentation name for this package.
	instrumentationName = "github.com/storageos/operator/controllers/webhook"

	// storageosclusterWebhookName is the name of the webhook controller for
	// storageoscluster resource.
	storageosclusterWebhookName = "storageoscluster-webhook"
)

var instrumentation *telemetry.Instrumentation

func init() {
	// Setup package instrumentation.
	instrumentation = telemetry.NewInstrumentationWithProviders(
		instrumentationName, nil, nil,
		ctrl.Log.WithName(storageosclusterWebhookName),
	)
}

// StorageOSClusterWebhook is the webhook controller for StorageOSCluster.
type StorageOSClusterWebhook struct {
	CtrlName string
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Client   client.Client

	singletonGetInstance singleton.GetInstanceFunc
}

var _ tkadmission.Controller = &StorageOSClusterWebhook{}

// NewStorageOSClusterWebhook constructs a webhook controller and returns it.
func NewStorageOSClusterWebhook(cl client.Client, scheme *runtime.Scheme) (*StorageOSClusterWebhook, error) {
	_, _, _, log := instrumentation.Start(context.Background(), "NewStorageOSClusterWebhook")

	// Instantiate the singleton function for the target object and set it in
	// the controller.
	getInstance, err := singleton.GetInstance(&storageoscomv1.StorageOSClusterList{}, scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to create singleton GetInstance: %w", err)
	}
	return &StorageOSClusterWebhook{
		CtrlName:             storageosclusterWebhookName,
		Scheme:               scheme,
		Log:                  log,
		Client:               cl,
		singletonGetInstance: getInstance,
	}, nil
}

// Name implements the admission webhook controller interface. It returns the
// webhook controller's name.
func (wh *StorageOSClusterWebhook) Name() string {
	return wh.CtrlName
}

// GetNewObject implements the admission webhook controller interface. It
// returns an instance of the target object.
func (wh *StorageOSClusterWebhook) GetNewObject() client.Object {
	return &storageoscomv1.StorageOSCluster{}
}

// RequireDefaulting implements the admission webhook controller interface. It
// is used to toggle the defaulter webhook.
func (wh *StorageOSClusterWebhook) RequireDefaulting(obj client.Object) bool {
	// Perform any relevant checks to determine if the object should be
	// defaulted or ignored.
	return false
}

// RequireValidating implements the admission webhook controller interface. It
// is used to toggle the validator webhook .
func (wh *StorageOSClusterWebhook) RequireValidating(obj client.Object) bool {
	// Perform any relevant checks to determine if the object should be
	// validated or ignored.
	return true
}

// Default implements the admission webhook controller interface. It returns a
// list of defaulter functions.
func (wh *StorageOSClusterWebhook) Default() []tkadmission.DefaultFunc {
	return []tkadmission.DefaultFunc{}
}

// ValidateCreate implements the admission webhook controller interface. It
// returns a list of validate on create functions.
func (wh *StorageOSClusterWebhook) ValidateCreate() []tkadmission.ValidateCreateFunc {
	return []tkadmission.ValidateCreateFunc{
		function.ValidateSingletonCreate(wh.singletonGetInstance, wh.Client),
	}
}

// ValidateUpdate implements the admission webhook controller interface. It
// returns a list of validate on update functions.
func (wh *StorageOSClusterWebhook) ValidateUpdate() []tkadmission.ValidateUpdateFunc {
	return []tkadmission.ValidateUpdateFunc{}
}

// ValidateDelete implements the admission webhook controller interface. It
// returns a list of validate on delete functions.
func (wh *StorageOSClusterWebhook) ValidateDelete() []tkadmission.ValidateDeleteFunc {
	return []tkadmission.ValidateDeleteFunc{}
}

// SetupWithManager builds the webhook controller, registering the webhook
// endpoints with the webhook server in the controller manager.
func (wh *StorageOSClusterWebhook) SetupWithManager(mgr manager.Manager) error {
	return builder.WebhookManagedBy(mgr).
		ValidatePath("/validate-storageoscluster").
		Complete(wh)
}
