/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"github.com/kanzifucius/promethues-operator-slos/pkg/slo"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	monitoringv1alpha1 "github.com/kanzifucius/promethues-operator-slos/api/v1alpha1"
	promoperator "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

const sloFinalizer = "slo.monitoring.kanzifucius.com"

// SloReconciler reconciles a Slo object
type SloReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=monitoring.kanzifucius.com,resources=sloes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.kanzifucius.com,resources=sloes/status,verbs=get;update;patch

func (r *SloReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("slo", req.NamespacedName)
	log.Info("Reconciling Memcached")
	ctx := context.Background()
	_ = r.Log.WithValues("slo", req.NamespacedName)

	sloDefinition := &monitoringv1alpha1.Slo{}
	err := r.Get(ctx, req.NamespacedName, sloDefinition)

	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Slo resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "error reading slo definition")
		return ctrl.Result{}, err
	}

	isSloToBeDeleted := sloDefinition.GetDeletionTimestamp() != nil
	if isSloToBeDeleted {
		if contains(sloDefinition.GetFinalizers(), sloFinalizer) {

			if err := r.addFinalizer(log, sloDefinition); err != nil {
				return ctrl.Result{}, err
			}
			// Remove Finalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(sloDefinition, sloFinalizer)
			err := r.Update(ctx, sloDefinition)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	found := &promoperator.PrometheusRule{}
	err = r.Get(ctx, types.NamespacedName{Name: sloDefinition.Name, Namespace: sloDefinition.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		rule, err := slo.GeneratePromRules(sloDefinition)
		if err != nil {
			log.Error(err, "Failed to generate Prometheus rule ")
			return ctrl.Result{}, err
		}
		err = ctrl.SetControllerReference(sloDefinition, rule, r.Scheme)
		if err != nil {
			log.Error(err, "Failed to set owner for Prometheus rule")
			return ctrl.Result{}, err
		}
		log.Info("Creating a new Prometheus Rules 	", "rule", rule.Name)
		err = r.Create(ctx, rule)
		if err != nil {
			r.Log.Error(err, "Failed to create new Prometheus")

			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Slo")
		return ctrl.Result{}, err
	}

	// check if we need to update the rule
	rule, err := slo.GeneratePromRules(sloDefinition)
	if !reflect.DeepEqual(found.Spec, rule.Spec) {
		found.Spec = rule.Spec
		err = ctrl.SetControllerReference(sloDefinition, rule, r.Scheme)
		if err != nil {
			log.Error(err, "Failed to update owner for Prometheus rule")
			return ctrl.Result{}, err
		}
		log.Info("Creating a new Prometheus Rules 	", "rule", rule.Name)
		err = r.Update(ctx, found)
		if err != nil {
			r.Log.Error(err, "Failed to update new Prometheus")

			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

func (r *SloReconciler) finalizeSLO(reqLogger logr.Logger, monitoringv1alpha1Slo *monitoringv1alpha1.Slo) error {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.
	reqLogger.Info("Successfully finalized slo")
	return nil
}

func (r *SloReconciler) addFinalizer(reqLogger logr.Logger, monitoringv1alpha1Slo *monitoringv1alpha1.Slo) error {
	reqLogger.Info("Adding Finalizer for the slo")
	controllerutil.AddFinalizer(monitoringv1alpha1Slo, sloFinalizer)

	// Update CR
	err := r.Update(context.TODO(), monitoringv1alpha1Slo)
	if err != nil {
		reqLogger.Error(err, "Failed to update slo with finalizer")
		return err
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func (r *SloReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1alpha1.Slo{}).
		Owns(&promoperator.PrometheusRule{}).
		Complete(r)
}
