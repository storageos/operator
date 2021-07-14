package storageoscluster

import (
	"context"

	"github.com/darkowlzz/operator-toolkit/declarative"
	"github.com/darkowlzz/operator-toolkit/declarative/kubectl"
	eventv1 "github.com/darkowlzz/operator-toolkit/event/v1"
	"github.com/darkowlzz/operator-toolkit/operator/v1/operand"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/filesys"
)

// afterInstallPackage contains the resource manifests for afterInstall
// operand.
const afterInstallPackage = "after-install"

type AfterInstallOperand struct {
	name            string
	client          client.Client
	requires        []string
	requeueStrategy operand.RequeueStrategy
	fs              filesys.FileSystem
	kubectlClient   kubectl.KubectlClient
}

var _ operand.Operand = &AfterInstallOperand{}

func (ai *AfterInstallOperand) Name() string                             { return ai.name }
func (ai *AfterInstallOperand) Requires() []string                       { return ai.requires }
func (ai *AfterInstallOperand) RequeueStrategy() operand.RequeueStrategy { return ai.requeueStrategy }
func (ai *AfterInstallOperand) ReadyCheck(ctx context.Context, obj client.Object) (bool, error) {
	return true, nil
}
func (c *AfterInstallOperand) PostReady(ctx context.Context, obj client.Object) error { return nil }

func (ai *AfterInstallOperand) Ensure(ctx context.Context, obj client.Object, ownerRef metav1.OwnerReference) (eventv1.ReconcilerEvent, error) {
	ctx, span, _, _ := instrumentation.Start(ctx, "AfterInstallOperand.Ensure")
	defer span.End()

	b, err := getAfterInstallBuilder(ai.fs, obj, ai.kubectlClient)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return nil, b.Apply(ctx)
}

func (ai *AfterInstallOperand) Delete(ctx context.Context, obj client.Object) (eventv1.ReconcilerEvent, error) {
	ctx, span, _, _ := instrumentation.Start(ctx, "AfterInstallOperand.Delete")
	defer span.End()

	b, err := getAfterInstallBuilder(ai.fs, obj, ai.kubectlClient)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return nil, b.Delete(ctx)
}

func getAfterInstallBuilder(fs filesys.FileSystem, obj client.Object, kcl kubectl.KubectlClient) (*declarative.Builder, error) {
	return declarative.NewBuilder(afterInstallPackage, fs,
		declarative.WithKubectlClient(kcl),
	)
}

func NewAfterInstallOperand(
	name string,
	client client.Client,
	requires []string,
	requeueStrategy operand.RequeueStrategy,
	fs filesys.FileSystem,
	kcl kubectl.KubectlClient,
) *AfterInstallOperand {
	return &AfterInstallOperand{
		name:            name,
		client:          client,
		requires:        requires,
		requeueStrategy: requeueStrategy,
		fs:              fs,
		kubectlClient:   kcl,
	}
}
