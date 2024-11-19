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
	"fmt"
	"os"

	"github.com/konflux-ci/release-service/loader"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// PipelineRunCoordinator is responsible for creating and managing renovate pipelineruns
type PipelinerunCoordinator struct {
	renovateImageUrl string
	debug            bool
	client           client.Client
	scheme           *runtime.Scheme
}

// PipelineRunReconciler reconciles a PipelineRun object
type PipelineRunReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewPipelinerunCoordinator(client client.Client, scheme *runtime.Scheme) *PipelinerunCoordinator {
	renovateImageUrl := os.Getenv(RenovateImageEnvName)
	if renovateImageUrl == "" {
		renovateImageUrl = DefaultRenovateImageUrl
	}
	return &PipelinerunCoordinator{renovateImageUrl: renovateImageUrl, client: client, scheme: scheme, debug: true}
}

// createPipelineRun creates and returns a new PipelineRun
// TODO: I need to add annotations, labels etc... with funcs like "WithAnnotations"
func (p *PipelinerunCoordinator) createPipelineRun(resources *loader.ProcessingResources) (*tektonv1.PipelineRun, error) {

	// Creating the pipelineRun definition
	pipelineRun := &tektonv1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			// TODO: update name
			Name:      "pipelinerun-test",
			Namespace: client.InNamespace(MintMakerNamespaceName),
		},
		Spec: tektonv1beta1.PipelineRunSpec{
			PipelineSpec: &tektonv1beta1.PipelineSpec{
				Tasks: []tektonv1beta1.PipelineTask{
					{
						Name: "build",
						TaskSpec: &tektonv1beta1.EmbeddedTask{
							TaskSpec: tektonv1beta1.TaskSpec{
								Steps: []tektonv1beta1.Step{
									{
										// TODO: this of course needs to be updated
										Name:  "renovate",
										Image: "alpine",
										Script: `
	                                    echo "Running Renovate"
	                                    sleep 10
	                                `,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := p.Client.Create(ctx, pipelineRun); err != nil {
		return nil, err
	}

	return pipelineRun, nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PipelineRun object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *PipelineRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	logger := log.FromContext(ctx)

	/* TODO: in CWFHEALTH-3296 we'll change the reconcile to also recognize different events. This is just
	 catching the creation of the pipelinerun. It will have to do something different when the pipelinerun is
	updated or finishes its run. */
	logger.Info(fmt.Sprintf("Created pipelinerun %v", req.NamespacedName))
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PipelineRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tektonv1beta1.PipelineRun{}).
		WithEventFilter(predicate.Funcs{
			DeleteFunc:  func(deleteEvent event.DeleteEvent) bool { return false },
			UpdateFunc:  func(updateEvent event.UpdateEvent) bool { return false },
			GenericFunc: func(genericEvent event.GenericEvent) bool { return false },
		}).
		Complete(r)
}
