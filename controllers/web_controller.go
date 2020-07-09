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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"

	webv1 "helloworld/m/api/v1"
)

// WebReconciler reconciles a Web object
type WebReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=web.helloworld,resources=webs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=web.helloworld,resources=webs/status,verbs=get;update;patch

func (r *WebReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("web", req.NamespacedName)

	// your logic here
	web := webv1.Web{}
	err := r.Get(ctx, req.NamespacedName, &web)
	if err != nil && errors.IsNotFound(err) {
		log.Info("web not found")
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "fail to get web")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	webType := web.Spec.Type
	groupVersionKind := web.GroupVersionKind()

	log.Info("get web success", "type", webType, "gvk", groupVersionKind)
	cr := unstructured.Unstructured{}
	cr.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   groupVersionKind.Group,
		Version: groupVersionKind.Version,
		Kind:    webType,
	})
	err = r.Get(ctx, req.NamespacedName, &cr)
	if err != nil && errors.IsNotFound(err) {
		log.Info("associate cr not found", "type", webType)
		newCr, err := r.newCr(&web)
		if err != nil {
			log.Error(err, "fail to init new cr", "type", webType)
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		err = r.Create(ctx, newCr)
		if err != nil {
			log.Error(err, "fail to create cr", "type", webType)
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		log.Info("create cr success", "type", webType)
	} else if err != nil {
		log.Error(err, "fail to find cr", "type", webType)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return ctrl.Result{}, nil
}

func (r *WebReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webv1.Web{}).
		Complete(r)
}

func (r *WebReconciler) newCr(web *webv1.Web) (*unstructured.Unstructured, error) {
	cr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": web.Spec.Spec,
		},
	}
	gvk := web.GroupVersionKind()
	cr.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    web.Spec.Type,
	})
	cr.SetName(web.Name + "-" + strings.ToLower(web.Spec.Type))
	cr.SetNamespace(web.Namespace)

	err := controllerutil.SetControllerReference(web, cr, r.Scheme)
	if err != nil {
		return nil, err
	} else {
		return cr, nil
	}
}
