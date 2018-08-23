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

package podpresetbinding

import (
	"context"
	"fmt"
	"strconv"

	"github.com/golang/glog"
	settingsv1alpha1 "github.com/jpeeler/podpreset-crd/pkg/apis/settings/v1alpha1"
	podpresetv1alpha1 "github.com/jpeeler/podpresetbinding-crd/pkg/apis/podpreset/v1alpha1"
	servicecatalogv1beta1 "github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new PodPresetBinding Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
// USER ACTION REQUIRED: update cmd/manager/main.go to call this podpreset.Add(mgr) to install this Controller
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePodPresetBinding{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("podpresetbinding-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to PodPresetBinding
	err = c.Watch(&source.Kind{Type: &podpresetv1alpha1.PodPresetBinding{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create
	// Uncomment watch a Deployment created by PodPresetBinding - change this for objects you create
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &podpresetv1alpha1.PodPresetBinding{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcilePodPresetBinding{}

// ReconcilePodPresetBinding reconciles a PodPresetBinding object
type ReconcilePodPresetBinding struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a PodPresetBinding object and makes changes based on the state read
// and what is in the PodPresetBinding.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  The scaffolding writes
// a Deployment as an example
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=podpreset.svcat.k8s.io,resources=podpresetbindings,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcilePodPresetBinding) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	glog.V(6).Infof("Entering reconcile: %#v", request)
	// Fetch the PodPresetBinding instance
	instance := &podpresetv1alpha1.PodPresetBinding{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Fetch the referenced ServiceBinding
	binding := &servicecatalogv1beta1.ServiceBinding{}
	glog.V(6).Infof("Instance %#v\n", instance)
	if instance.Spec.BindingRef == nil {
		glog.V(6).Infof("BindingRef was nil, bailing\n")
		return reconcile.Result{}, fmt.Errorf("spec for instance '%v' did not contain bindingref", instance)
	}
	err = r.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.BindingRef.Name, Namespace: instance.Namespace}, binding)
	glog.V(6).Infof("Got: %#v binding:%#v", err, binding)

	if err != nil {
		if errors.IsNotFound(err) {
			glog.V(6).Infof("Binding '%v' not found, requeuing", instance.Spec.BindingRef.Name)
		}
		// error reading the object - requeue the request
		return reconcile.Result{}, err
	}

	glog.V(6).Infof("Looking at binding %+v\n", binding)
	if len(binding.Status.Conditions) == 0 {
		// binding not ready, requeue the request
		return reconcile.Result{}, fmt.Errorf("Binding '%v' not yet ready, requeuing", binding.Name)
	} else if binding.Status.Conditions[len(binding.Status.Conditions)-1].Type == servicecatalogv1beta1.ServiceBindingConditionReady {
		// create pod preset if binding status is ready and it doesn't already exist

		podPresetList := &settingsv1alpha1.PodPresetList{}
		err = r.List(context.TODO(), &client.ListOptions{Namespace: instance.Namespace}, podPresetList)
		if err != nil {
			return reconcile.Result{}, err
		}
		for _, podpreset := range podPresetList.Items {
			glog.V(6).Infof("Looking at podpreset %v\n", podpreset)
			for _, ownerRefs := range podpreset.OwnerReferences {
				if ownerRefs.UID == instance.UID {
					instanceResourceVersion, err := strconv.Atoi(instance.ResourceVersion)
					if err != nil {
						return reconcile.Result{}, err
					}
					podpresetResourceVersion, err := strconv.Atoi(podpreset.ResourceVersion)
					if err != nil {
						return reconcile.Result{}, err
					}

					if instanceResourceVersion < podpresetResourceVersion {
						glog.V(6).Info("Found existing podpreset and no update required")
						return reconcile.Result{}, nil
					}
					glog.V(6).Info("Found existing podpreset and update required")
					podpreset.Spec = instance.Spec.PodPresetTemplate.Spec
					if err = r.Update(context.TODO(), &podpreset); err != nil {
						glog.V(2).Infof("Error during update of podpreset %v\n", err)
						return reconcile.Result{}, err
					}
					return reconcile.Result{}, nil
				}
			}
		}
		newPodPreset := settingsv1alpha1.PodPreset{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "autogenerated-podpreset-",
				Namespace:    instance.Namespace,
			},
			Spec: instance.Spec.PodPresetTemplate.Spec,
		}
		if err := controllerutil.SetControllerReference(instance, &newPodPreset, r.scheme); err != nil {
			return reconcile.Result{}, err
		}
		glog.V(6).Infof("Binding '%v' ready, attempting to create pod preset %#v\n", binding.Name, newPodPreset)
		if err := r.Create(context.TODO(), &newPodPreset); err != nil {
			glog.V(5).Infof("Podpreset creation failed: %v\n", err)
			return reconcile.Result{}, err
		}
		glog.V(6).Infof("Create for pod preset %v was successful!\n", newPodPreset.Name)
	}
	return reconcile.Result{}, nil
}
