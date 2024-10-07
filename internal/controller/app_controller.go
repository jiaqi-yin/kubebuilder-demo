/*
Copyright 2024.

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

package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ingressv1beta1 "github.com/jiaqi-yin/kubebuilder-demo/api/v1beta1"
	"github.com/jiaqi-yin/kubebuilder-demo/internal/controller/utils"
)

// AppReconciler reconciles a App object
type AppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=ingress.kb.dev,resources=apps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ingress.kb.dev,resources=apps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ingress.kb.dev,resources=apps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the App object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *AppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	app := &ingressv1beta1.App{}

	// Get app from cache.
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 1. Handle deployment.
	deployment := utils.NewDeployment(app)
	if err := controllerutil.SetControllerReference(app, deployment, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	d := &appsv1.Deployment{}
	if err := r.Get(ctx, req.NamespacedName, d); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("deployment not found")
			if err := r.Create(ctx, deployment); err != nil {
				logger.Error(err, "create deploy failed")
				return ctrl.Result{}, err
			}
		}
	} else {
		if app.Spec.Replicas != *d.Spec.Replicas && app.Spec.Image != d.Spec.Template.Spec.Containers[0].Image {
			if err := r.Update(ctx, deployment); err != nil {
				logger.Info("update deploy")
				return ctrl.Result{}, err
			}
		}
	}

	// 2. Handle service.
	service := utils.NewService(app)
	if err := controllerutil.SetControllerReference(app, service, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	s := &corev1.Service{}
	if err := r.Get(ctx, req.NamespacedName, s); err != nil {
		if errors.IsNotFound(err) && app.Spec.EnableService {
			logger.Info("service not found and enable_service is true")
			if err := r.Create(ctx, service); err != nil {
				logger.Error(err, "create service failed")
				return ctrl.Result{}, err
			}
		}
		if !errors.IsNotFound(err) && app.Spec.EnableService {
			return ctrl.Result{}, err
		}
	} else {
		if app.Spec.EnableService {
			logger.Info("skip update service")
		} else {
			if err := r.Delete(ctx, s); err != nil {
				return ctrl.Result{}, err
			}

		}
	}

	//3. Handle ingress.
	ingress := utils.NewIngress(app)
	if err := controllerutil.SetControllerReference(app, ingress, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	i := &netv1.Ingress{}
	if err := r.Get(ctx, req.NamespacedName, i); err != nil {
		if errors.IsNotFound(err) && app.Spec.EnableIngress {
			logger.Info("ingress not found and enable_ingress is true")
			if err := r.Create(ctx, ingress); err != nil {
				logger.Error(err, "create ingress failed")
				return ctrl.Result{}, err
			}
		}
		if !errors.IsNotFound(err) && app.Spec.EnableIngress {
			return ctrl.Result{}, err
		}
	} else {
		if app.Spec.EnableIngress {
			logger.Info("skip update ingress")
		} else {
			if err := r.Delete(ctx, i); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ingressv1beta1.App{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&netv1.Ingress{}).
		Complete(r)
}
