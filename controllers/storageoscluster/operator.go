package storageoscluster

import (
	"fmt"

	operatorv1 "github.com/darkowlzz/operator-toolkit/operator/v1"
	"github.com/darkowlzz/operator-toolkit/operator/v1/executor"
	"github.com/darkowlzz/operator-toolkit/operator/v1/operand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/kustomize/api/filesys"
)

const (
	apiManagerOpName = "api-manager-operand"
	csiOpName        = "csi-operand"
	schedulerOpName  = "scheduler-operand"
	nodeOpName       = "node-operand"
)

func NewOperator(mgr ctrl.Manager, fs filesys.FileSystem, execStrategy executor.ExecutionStrategy) (*operatorv1.CompositeOperator, error) {
	// Create operands with their relationships.
	//
	//          +----+         +---------+
	//       +--+Node+----+    |Scheduler|
	//       |  +----+    |    +---------+
	//       |            |
	//       |            |
	//       v            v
	//     +---+     +-----------+
	//     |CSI|     |api-manager|
	//     +---+     +-----------+
	//
	// CSI and api-manager operands depend on Node. Node and Scheduler are
	// independent.
	apiManagerOp := NewAPIManagerOperand(apiManagerOpName, mgr.GetClient(), []string{nodeOpName}, operand.RequeueOnError, fs)
	csiOp := NewCSIOperand(csiOpName, mgr.GetClient(), []string{nodeOpName}, operand.RequeueOnError, fs)
	schedulerOp := NewSchedulerOperand(schedulerOpName, mgr.GetClient(), []string{}, operand.RequeueOnError, fs)
	nodeOp := NewNodeOperand(nodeOpName, mgr.GetClient(), []string{}, operand.RequeueOnError, fs)

	// Create and return CompositeOperator.
	return operatorv1.NewCompositeOperator(
		operatorv1.WithEventRecorder(mgr.GetEventRecorderFor("storageoscluster-controller")),
		operatorv1.WithExecutionStrategy(execStrategy),
		operatorv1.WithOperands(apiManagerOp, csiOp, schedulerOp, nodeOp),
	)
}

func NewStorageOSClusterController(mgr ctrl.Manager, fs filesys.FileSystem, execStrategy executor.ExecutionStrategy) (*StorageOSClusterController, error) {
	operator, err := NewOperator(mgr, fs, execStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new operator: %w", err)
	}
	return &StorageOSClusterController{Operator: operator, Client: mgr.GetClient()}, nil
}