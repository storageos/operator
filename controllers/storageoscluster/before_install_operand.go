package storageoscluster

import (
	"context"

	"github.com/darkowlzz/operator-toolkit/declarative"
	eventv1 "github.com/darkowlzz/operator-toolkit/event/v1"
	"github.com/darkowlzz/operator-toolkit/operator/v1/operand"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/filesys"
)

// beforeInstallPackage contains the resource manifests for beforeInstall
// operand.
const beforeInstallPackage = "before-install"

type BeforeInstallOperand struct {
	name            string
	client          client.Client
	requires        []string
	requeueStrategy operand.RequeueStrategy
	fs              filesys.FileSystem
}

var _ operand.Operand = &BeforeInstallOperand{}

func (bi *BeforeInstallOperand) Name() string                             { return bi.name }
func (bi *BeforeInstallOperand) Requires() []string                       { return bi.requires }
func (bi *BeforeInstallOperand) RequeueStrategy() operand.RequeueStrategy { return bi.requeueStrategy }
func (bi *BeforeInstallOperand) ReadyCheck(ctx context.Context, obj client.Object) (bool, error) {
	return true, nil
}
func (c *BeforeInstallOperand) PostReady(ctx context.Context, obj client.Object) error { return nil }

func (bi *BeforeInstallOperand) Ensure(ctx context.Context, obj client.Object, ownerRef metav1.OwnerReference) (eventv1.ReconcilerEvent, error) {
	ctx, span, _, _ := instrumentation.Start(ctx, "BeforeInstallOperand.Ensure")
	defer span.End()

	b, err := getBeforeInstallBuilder(bi.fs, obj)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return nil, b.Apply(ctx)
}

func (bi *BeforeInstallOperand) Delete(ctx context.Context, obj client.Object) (eventv1.ReconcilerEvent, error) {
	ctx, span, _, _ := instrumentation.Start(ctx, "BeforeInstallOperand.Delete")
	defer span.End()

	b, err := getBeforeInstallBuilder(bi.fs, obj)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return nil, b.Delete(ctx)
}

func getBeforeInstallBuilder(fs filesys.FileSystem, obj client.Object) (*declarative.Builder, error) {
	return declarative.NewBuilder(beforeInstallPackage, fs)
}

func NewBeforeInstallOperand(
	name string,
	client client.Client,
	requires []string,
	requeueStrategy operand.RequeueStrategy,
	fs filesys.FileSystem,
) *BeforeInstallOperand {
	return &BeforeInstallOperand{
		name:            name,
		client:          client,
		requires:        requires,
		requeueStrategy: requeueStrategy,
		fs:              fs,
	}
}
