package storageoscluster

import (
	"context"
	"fmt"

	compositev1 "github.com/darkowlzz/operator-toolkit/controller/composite/v1"
	"github.com/darkowlzz/operator-toolkit/object"
	operatorv1 "github.com/darkowlzz/operator-toolkit/operator/v1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storageoscomv1 "github.com/storageos/operator/apis/v1"
)

const (
	schedulerReadyType  = "SchedulerReady"
	nodeReadyType       = "NodeReady"
	apiManagerReadyType = "APIManagerReady"
	csiReadyType        = "CSIReady"

	readyReason    = "Ready"
	notReadyReason = "NotReady"
)

type StorageOSClusterController struct {
	Operator operatorv1.Operator
	Client   client.Client
}

var _ compositev1.Controller = &StorageOSClusterController{}

func (c *StorageOSClusterController) Default(context.Context, client.Object) {}

func (c *StorageOSClusterController) Validate(context.Context, client.Object) error { return nil }

func (c *StorageOSClusterController) Initialize(ctx context.Context, obj client.Object, condn metav1.Condition) error {
	_, span, _, _ := instrumentation.Start(ctx, "StorageOSClusterController.Initialize")
	defer span.End()

	cluster, ok := obj.(*storageoscomv1.StorageOSCluster)
	if !ok {
		return fmt.Errorf("failed to convert %v to StorageOSCluster", obj)
	}

	meta.SetStatusCondition(&cluster.Status.Conditions, condn)
	span.AddEvent("Added initial condition to status")

	return nil
}

func (c *StorageOSClusterController) Operate(ctx context.Context, obj client.Object) (result ctrl.Result, err error) {
	return c.Operator.Ensure(ctx, obj, object.OwnerReferenceFromObject(obj))
}

func (c *StorageOSClusterController) Cleanup(ctx context.Context, obj client.Object) (result ctrl.Result, err error) {
	return c.Operator.Cleanup(ctx, obj)
}

func (c *StorageOSClusterController) UpdateStatus(ctx context.Context, obj client.Object) error {
	ctx, span, _, log := instrumentation.Start(ctx, "StorageOSClusterController.UpdateStatus")
	defer span.End()

	cluster, ok := obj.(*storageoscomv1.StorageOSCluster)
	if !ok {
		return fmt.Errorf("failed to convert %v to StorageOSCluster", obj)
	}

	// Get the latest version of the cluster.
	if getErr := c.Client.Get(ctx, client.ObjectKeyFromObject(cluster), cluster); getErr != nil {
		if apierrors.IsNotFound(getErr) {
			// If the object is not found, the object may have been deleted by
			// cleanup handler call before UpdateStatus was called.
			return nil
		}
		return fmt.Errorf("failed to get StorageOSCluster %q: %w", cluster.GetName(), getErr)
	}
	span.AddEvent("Fetched new instance")

	// Check status of all the components.

	schedulerCondition := getSchedulerCondition(ctx, c.Client, obj.GetNamespace(), log)
	meta.SetStatusCondition(&cluster.Status.Conditions, schedulerCondition)

	nodeCondition := getNodeCondition(ctx, c.Client, obj.GetNamespace(), log)
	meta.SetStatusCondition(&cluster.Status.Conditions, nodeCondition)

	apiManagerCondition := getAPIManagerCondition(ctx, c.Client, obj.GetNamespace(), log)
	meta.SetStatusCondition(&cluster.Status.Conditions, apiManagerCondition)

	csiCondition := getCSICondition(ctx, c.Client, obj.GetNamespace(), log)
	meta.SetStatusCondition(&cluster.Status.Conditions, csiCondition)

	conditions := cluster.Status.Conditions

	// Evaluate the cluster phase based on the component status.
	phase := "Pending"
	if meta.IsStatusConditionTrue(conditions, schedulerReadyType) ||
		meta.IsStatusConditionTrue(conditions, nodeReadyType) ||
		meta.IsStatusConditionTrue(conditions, apiManagerReadyType) ||
		meta.IsStatusConditionTrue(conditions, csiReadyType) {
		// Some components are ready, Creating phase.
		phase = "Creating"
	}

	// Evaluate the cluster condition based on the component status.
	if meta.IsStatusConditionTrue(conditions, schedulerReadyType) &&
		meta.IsStatusConditionTrue(conditions, nodeReadyType) &&
		meta.IsStatusConditionTrue(conditions, apiManagerReadyType) &&
		meta.IsStatusConditionTrue(conditions, csiReadyType) {
		// Remove progressing condition and set cluster Ready status.
		meta.RemoveStatusCondition(&cluster.Status.Conditions, "Progressing")
		readyCondition := metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionTrue,
			Reason:  "Ready",
			Message: "Cluster Ready",
		}
		meta.SetStatusCondition(&cluster.Status.Conditions, readyCondition)

		// All the components are ready, Running phase.
		phase = "Running"
	}

	// Set the cluster phase.
	cluster.Status.Phase = phase

	// Get the control-plane instances and set them in the members status.
	members, err := getControlPlaneMembers(ctx, c.Client, obj.GetNamespace(), log)
	if err != nil {
		return err
	}
	cluster.Status.Members = members

	// Populate the ready value from members status.
	cluster.Status.Ready = getReadyFromMembersStatus(members)

	return nil
}

func getSchedulerCondition(ctx context.Context, cl client.Client, namespace string, log logr.Logger) metav1.Condition {
	schedulerCondition := metav1.Condition{
		Type:    schedulerReadyType,
		Status:  metav1.ConditionFalse,
		Reason:  notReadyReason,
		Message: "Scheduler Not Ready",
	}
	schedulerDep := &appsv1.Deployment{}
	schedulerKey := client.ObjectKey{Name: "storageos-scheduler", Namespace: namespace}
	if err := cl.Get(ctx, schedulerKey, schedulerDep); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "failed to get storageos-scheduler status")
		}
	}
	if schedulerDep.Status.ReadyReplicas == schedulerDep.Status.Replicas {
		schedulerCondition.Status = metav1.ConditionTrue
		schedulerCondition.Reason = readyReason
		schedulerCondition.Message = "Scheduler Ready"
	}
	return schedulerCondition
}

func getNodeCondition(ctx context.Context, cl client.Client, namespace string, log logr.Logger) metav1.Condition {
	nodeCondition := metav1.Condition{
		Type:    nodeReadyType,
		Status:  metav1.ConditionFalse,
		Reason:  notReadyReason,
		Message: "Node Not Ready",
	}
	nodeDS := &appsv1.DaemonSet{}
	nodeKey := client.ObjectKey{Name: "storageos-daemonset", Namespace: namespace}
	if err := cl.Get(ctx, nodeKey, nodeDS); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "failed to get node status")
		}
	}
	if nodeDS.Status.NumberReady > 0 {
		nodeCondition.Status = metav1.ConditionTrue
		nodeCondition.Reason = readyReason
		nodeCondition.Message = "Node Ready"
	}
	return nodeCondition
}

func getAPIManagerCondition(ctx context.Context, cl client.Client, namespace string, log logr.Logger) metav1.Condition {
	apiManagerCondition := metav1.Condition{
		Type:    apiManagerReadyType,
		Status:  metav1.ConditionFalse,
		Reason:  notReadyReason,
		Message: "APIManager Not Ready",
	}
	amDep := &appsv1.Deployment{}
	amKey := client.ObjectKey{Name: "storageos-api-manager", Namespace: namespace}
	if err := cl.Get(ctx, amKey, amDep); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "failed to get api-manager status")
		}
	}
	if amDep.Status.AvailableReplicas > 0 {
		apiManagerCondition.Status = metav1.ConditionTrue
		apiManagerCondition.Reason = readyReason
		apiManagerCondition.Message = "APIManager Ready"
	}
	return apiManagerCondition
}

func getCSICondition(ctx context.Context, cl client.Client, namespace string, log logr.Logger) metav1.Condition {
	csiCondition := metav1.Condition{
		Type:    csiReadyType,
		Status:  metav1.ConditionFalse,
		Reason:  notReadyReason,
		Message: "CSI Not Ready",
	}
	csiDep := &appsv1.Deployment{}
	csiKey := client.ObjectKey{Name: "storageos-csi-helper", Namespace: namespace}
	if err := cl.Get(ctx, csiKey, csiDep); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "failed to get csi-helper status")
		}
	}
	if csiDep.Status.AvailableReplicas > 0 {
		csiCondition.Status = metav1.ConditionTrue
		csiCondition.Reason = readyReason
		csiCondition.Message = "CSI Ready"
	}
	return csiCondition
}

// getLabelsForControlPlane returns the labels for selecting storageos
// control-plane.
func getLabelsForControlPlane() map[string]string {
	return map[string]string{
		"app":                         "storageos",
		"app.kubernetes.io/component": "control-plane",
	}
}

// getReadyFromMembersStatus calculates the ready status of the cluster based
// on the member status.
func getReadyFromMembersStatus(m storageoscomv1.MembersStatus) string {
	return fmt.Sprintf("%d/%d", len(m.Ready), len(m.Ready)+len(m.Unready))
}

// getControlPlaneMembers fetches the storageos control-plane pods and returns
// a MembersStatus based on the pod status.
func getControlPlaneMembers(ctx context.Context, cl client.Client, namespace string, log logr.Logger) (storageoscomv1.MembersStatus, error) {
	ms := storageoscomv1.MembersStatus{}
	cpPods := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(getLabelsForControlPlane()),
	}
	if err := cl.List(ctx, cpPods, listOpts...); err != nil {
		log.Error(err, "failed to list control-plane pods", "namespace", namespace)
		return ms, err
	}

	for _, pod := range cpPods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			ms.Ready = append(ms.Ready, pod.Status.HostIP)
		} else {
			ms.Unready = append(ms.Unready, pod.Status.HostIP)
		}
	}

	return ms, nil
}
