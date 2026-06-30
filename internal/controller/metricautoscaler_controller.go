package controller

import (
	"context"
	"fmt"
	"strconv"
	"time"

	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	appsv1k8s "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "github.com/nkarthik23/k8s-custom-controller/api/v1"
)

// MetricAutoscalerReconciler reconciles a MetricAutoscaler object
type MetricAutoscalerReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	PrometheusURL  string
}

// +kubebuilder:rbac:groups=apps.yourdomain.dev,resources=metricautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.yourdomain.dev,resources=metricautoscalers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.yourdomain.dev,resources=metricautoscalers/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch

func (r *MetricAutoscalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Step 1: Fetch the MetricAutoscaler resource
	var autoscaler appsv1.MetricAutoscaler
	if err := r.Get(ctx, req.NamespacedName, &autoscaler); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Reconciling", "name", autoscaler.Name)

	// Step 2: Query Prometheus for the current metric value
	promClient, err := promapi.NewClient(promapi.Config{
		Address: r.PrometheusURL,
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	promAPI := promv1.NewAPI(promClient)
	result, warnings, err := promAPI.Query(ctx, autoscaler.Spec.PrometheusQuery, time.Now())
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to query Prometheus: %w", err)
	}
	if len(warnings) > 0 {
		log.Info("Prometheus warnings", "warnings", warnings)
	}

	// Step 3: Parse the metric value
	vector, ok := result.(model.Vector)
	if !ok || len(vector) == 0 {
		log.Info("No metric data returned, requeueing")
		return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
	}
	metricValue := float64(vector[0].Value)
	log.Info("Current metric value", "metric", autoscaler.Spec.PrometheusQuery, "value", metricValue)

	// Step 4: Parse threshold and calculate desired replicas
	threshold, err := strconv.ParseFloat(autoscaler.Spec.Threshold, 64)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("invalid threshold value: %w", err)
	}

	desiredReplicas := autoscaler.Spec.MinReplicas
	if metricValue > threshold {
		desiredReplicas = autoscaler.Spec.MaxReplicas
	}

	// Step 5: Fetch the target Deployment
	var deployment appsv1k8s.Deployment
	if err := r.Get(ctx, types.NamespacedName{
		Name:      autoscaler.Spec.TargetDeployment,
		Namespace: req.Namespace,
	}, &deployment); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get deployment: %w", err)
	}

	// Step 6: Update replicas if needed
	if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas != desiredReplicas {
		log.Info("Scaling deployment",
			"deployment", autoscaler.Spec.TargetDeployment,
			"from", deployment.Spec.Replicas,
			"to", desiredReplicas)
		deployment.Spec.Replicas = &desiredReplicas
		if err := r.Update(ctx, &deployment); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update deployment: %w", err)
		}
	}

	// Step 7: Requeue after 15 seconds to check again
	return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MetricAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.MetricAutoscaler{}).
		Named("metricautoscaler").
		Complete(r)
}