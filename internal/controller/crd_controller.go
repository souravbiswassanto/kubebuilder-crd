/*
Copyright 2023.

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
	"fmt"
	"github.com/souravbiswassanto/kubebuilder-crd/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	logger "log"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	"time"
)

// CrdReconciler reconciles a Crd object
type CrdReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=makecrd.com.makecrd.com,resources=crds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=makecrd.com.makecrd.com,resources=crds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=makecrd.com.makecrd.com,resources=crds/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Crd object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *CrdReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logs := log.FromContext(ctx)
	logs.WithValues("ReqName ", req.Name, "ReqNameSpace: ", req.Namespace)
	fmt.Println("Printing Req: ", req)
	// TODO(user): your logic here
	var crd v1alpha1.Crd

	if err := r.Get(ctx, req.NamespacedName, &crd); err != nil {
		logger.Println(err, "Unable to fectch crd resource")
		return ctrl.Result{}, nil
	}
	logger.Println("Resource found, Resource name ", crd.Name)
	var deploymentObject appsv1.Deployment
	logger.Println("NamespaceName ", req.Name, " ", req.Namespace, " ", crd.Spec.Name)
	objectKey := client.ObjectKey{
		Namespace: req.Namespace,
		Name:      crd.Spec.Name,
	}
	if objectKey.Name == "" {
		objectKey.Name = strings.Join(buildSlice(crd.Name, v1alpha1.Deployment), v1alpha1.Dash)
	}
	if err := r.Get(ctx, objectKey, &deploymentObject); err != nil {
		if errors.IsNotFound(err) {
			logger.Println("Deployment ", objectKey.Name, " is not found. Creating a new one ...")
			err := r.Create(ctx, newDeployment(&crd))
			if err != nil {
				logger.Printf("error while creating deployment %\n", crd.Name)
				return ctrl.Result{}, err
			}
			logger.Println("Created")
			return ctrl.Result{}, err
		}
		logger.Printf("error fetching deployment %s\n", err)
		return ctrl.Result{}, err
	} else {
		if crd.Spec.Replicas != nil && *crd.Spec.Replicas != *deploymentObject.Spec.Replicas {
			logger.Println(*crd.Spec.Replicas, *deploymentObject.Spec.Replicas)
			logger.Println("deployments replicas don't match ")
			*deploymentObject.Spec.Replicas = *crd.Spec.Replicas
			if err := r.Update(ctx, &deploymentObject); err != nil {
				logger.Println("error updating deployment %s\n", err)
				return ctrl.Result{}, err
			}
			logger.Println("deployment updated")
		}
		if crd.Spec.Replicas != nil && *deploymentObject.Spec.Replicas != crd.Status.AvailableReplicas {
			logger.Println(*crd.Spec.Replicas, crd.Status.AvailableReplicas)
			var deepCopy *v1alpha1.Crd
			deepCopy = crd.DeepCopy()
			deepCopy.Status.AvailableReplicas = *deploymentObject.Spec.Replicas
			if err := r.Status().Update(ctx, deepCopy); err != nil {
				logger.Printf("error updating status %s\n", err)
				return ctrl.Result{}, err
			}
			logger.Println("status updated")
		}

	}
	var serviceObject corev1.Service
	objectKey = client.ObjectKey{
		Namespace: req.Namespace,
		Name:      crd.Spec.Name,
	}
	logger.Println(objectKey.Namespace)
	logger.Println(req)

	if objectKey.Name == "" {
		objectKey.Name = strings.Join(buildSlice(crd.Name, v1alpha1.Service), v1alpha1.Dash)
	}
	if err := r.Get(ctx, objectKey, &serviceObject); err != nil {
		if errors.IsNotFound(err) {
			err := r.Create(ctx, newService(&crd))
			if err != nil {
				logger.Printf("error while creating Service %s\n", err)
				return ctrl.Result{}, err
			}
			logger.Println("Service created successfully")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err

	} else {
		if err := r.Update(ctx, &serviceObject); err != nil {
			logger.Println("Service update error")
			return ctrl.Result{}, err
		}
		logger.Println("service updated")
	}

	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: 1 * time.Minute,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CrdReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Crd{}).
		//Watches(&source.Kind{Type: &appsv1.Deployment{}}, handlerForDeployment).
		//	Watches(&source.Kind{Type: &appsv1.Deployment{}}, handlerForDeployment).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func buildSlice(s ...string) []string {
	var t []string
	for _, v := range s {
		t = append(t, v)
	}
	return t
}
