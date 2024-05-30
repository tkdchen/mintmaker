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
	"reflect"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	mmv1alpha1 "github.com/konflux-ci/mintmaker/api/v1alpha1"
	"github.com/konflux-ci/mintmaker/pkg/git"
	"github.com/konflux-ci/mintmaker/pkg/k8s"
	"github.com/konflux-ci/mintmaker/pkg/renovate"
	appstudiov1alpha1 "github.com/redhat-appstudio/application-api/api/v1alpha1"
)

// DependencyUpdateCheckReconciler reconciles a DependencyUpdateCheck object
type DependencyUpdateCheckReconciler struct {
	client         client.Client
	taskProviders  []renovate.TaskProvider
	eventRecorder  record.EventRecorder
	jobCoordinator *renovate.JobCoordinator
}

func NewDependencyUpdateCheckReconciler(client client.Client, scheme *runtime.Scheme, eventRecorder record.EventRecorder) *DependencyUpdateCheckReconciler {
	return &DependencyUpdateCheckReconciler{
		client: client,
		taskProviders: []renovate.TaskProvider{
			renovate.NewGithubAppRenovaterTaskProvider(k8s.NewGithubAppConfigReader(client, scheme, eventRecorder)),
			renovate.NewBasicAuthTaskProvider(k8s.NewGitCredentialProvider(client)),
		},
		eventRecorder:  eventRecorder,
		jobCoordinator: renovate.NewJobCoordinator(client, scheme),
	}
}

//+kubebuilder:rbac:groups=appstudio.redhat.com,resources=dependencyupdatechecks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=appstudio.redhat.com,resources=dependencyupdatechecks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=appstudio.redhat.com,resources=dependencyupdatechecks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DependencyUpdateCheckReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx).WithName("DependencyUpdateCheckController")
	ctx = ctrllog.IntoContext(ctx, log)

	// Get Components
	componentList := &appstudiov1alpha1.ComponentList{}
	if err := r.client.List(ctx, componentList, &client.ListOptions{}); err != nil {
		log.Error(err, "failed to list Components")
		return ctrl.Result{}, err
	}
	var scmComponents []*git.ScmComponent
	for _, component := range componentList.Items {
		gitProvider, err := getGitProvider(component)
		if err != nil {
			// component misconfiguration shouldn't prevent other components from being updated
			// deepcopy the component to avoid implicit memory aliasing in for loop
			r.eventRecorder.Event(component.DeepCopy(), "Warning", "ErrorComponentProviderInfo", err.Error())
			continue
		}

		scmComponent, err := git.NewScmComponent(gitProvider, component.Spec.Source.GitSource.URL, component.Spec.Source.GitSource.Revision, component.Name, component.Namespace)
		if err != nil {
			return ctrl.Result{}, err
		}
		scmComponents = append(scmComponents, scmComponent)
	}
	var tasks []*renovate.Task
	for _, taskProvider := range r.taskProviders {
		newTasks := taskProvider.GetNewTasks(ctx, scmComponents)
		log.Info("found new tasks", "tasks", len(newTasks), "provider", reflect.TypeOf(taskProvider).String())
		if len(newTasks) > 0 {
			tasks = append(tasks, newTasks...)
		}
	}

	log.Info("executing renovate tasks", "tasks", len(tasks))
	err := r.jobCoordinator.ExecuteWithLimits(ctx, tasks)
	if err != nil {
		log.Error(err, "failed to create a job")
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DependencyUpdateCheckReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mmv1alpha1.DependencyUpdateCheck{}, builder.WithPredicates(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return true
			},
			DeleteFunc: func(event.DeleteEvent) bool {
				return false
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return false
			},
			GenericFunc: func(event.GenericEvent) bool {
				return false
			},
		})).
		Complete(r)
}
