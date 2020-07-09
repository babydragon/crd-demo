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
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	webv1 "helloworld/m/api/v1"
	appsv1 "k8s.io/api/apps/v1"
)

// BlogReconciler reconciles a Blog object
type BlogReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=web.helloworld,resources=blogs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=web.helloworld,resources=blogs/status,verbs=get;update;patch

func (r *BlogReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	// your logic here
	ctx := context.Background()
	log := r.Log.WithValues("blog", req.NamespacedName)

	blog := &webv1.Blog{}
	err := r.Get(ctx, req.NamespacedName, blog)

	if err != nil && errors.IsNotFound(err) {
		log.Info("blog not found")
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "fail to get blog")

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	url := blog.Spec.Url
	log.Info("get blog success", "url", url)

	label := map[string]string{
		"server": blog.Name,
	}

	// 检查和创建deployment
	deployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		log.Info("deployment not found, start to create")

		newDeployment, err := r.newDeployment(blog, label)
		if err != nil {
			log.Error(err, "fail to init deployment")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		err = r.Create(ctx, newDeployment)
		if err != nil {
			log.Error(err, "fail to create deployment")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		log.Info("create deployment success")
	} else if err != nil {
		log.Error(err, "fail to get deployment")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	} else {
		log.Info("get deployment success",
			"Namespace", deployment.Namespace,
			"name", deployment.Name,
			"replicas", *deployment.Spec.Replicas)
	}

	// 这里开始创建service
	service := &v1.Service{}
	err = r.Get(ctx, types.NamespacedName{
		Namespace: service.Namespace,
		Name:      service.Name,
	}, service)

	if err != nil && errors.IsNotFound(err) {
		log.Info("service not found, start to create")
		newService, err := r.newService(blog, label)
		if err != nil {
			log.Error(err, "fail to init service")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		err = r.Create(ctx, newService)
		if err != nil {
			log.Error(err, "fail to create service")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		log.Info("create service success")
	} else if err != nil {
		log.Error(err, "fail to get service")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	} else {
		log.Info("get service success",
			"name", service.Name,
			"namespace", service.Namespace)
	}

	return ctrl.Result{}, nil
}

func (r *BlogReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webv1.Blog{}).
		Complete(r)
}

func (r *BlogReconciler) newDeployment(blog *webv1.Blog, label map[string]string) (*appsv1.Deployment, error) {
	replicas := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      blog.Name,
			Namespace: blog.Namespace,
			Labels:    label,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: label,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   blog.Name,
					Labels: label,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "nginx",
							Image: "nginx:alpine",
							Ports: []v1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	err := controllerutil.SetControllerReference(blog, deployment, r.Scheme)
	if err != nil {
		return nil, err
	}

	return deployment, nil
}

func (r *BlogReconciler) newService(blog *webv1.Blog, label map[string]string) (*v1.Service, error) {
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      blog.Name + "-service",
			Namespace: blog.Namespace,
			Labels:    label,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       "http",
					Port:       8080,
					TargetPort: intstr.FromInt(80),
				},
			},
			Selector: label,
			Type:     v1.ServiceTypeLoadBalancer,
		},
	}

	err := controllerutil.SetControllerReference(blog, service, r.Scheme)
	if err != nil {
		return nil, err
	}

	return service, nil
}
