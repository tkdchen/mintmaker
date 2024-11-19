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
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	appstudiov1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	mmv1alpha1 "github.com/konflux-ci/mintmaker/api/v1alpha1"
	. "github.com/konflux-ci/mintmaker/pkg/common"
	"github.com/konflux-ci/mintmaker/pkg/git"
	"github.com/konflux-ci/mintmaker/pkg/renovate"
	"github.com/konflux-ci/release-service/loader"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	RenovateImageEnvName    = "RENOVATE_IMAGE"
	DefaultRenovateImageUrl = "quay.io/konflux-ci/mintmaker-renovate-image:latest"
)

// DependencyUpdateCheckReconciler reconciles a DependencyUpdateCheck object
type DependencyUpdateCheckReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	renovateImageUrl string
}

// Create a secret that merges all secret with the label:
// mintmaker.appstudio.redhat.com/secret-type: registry
// and return the new secret
func (r *DependencyUpdateCheckReconciler) createMergedPullSecret(ctx context.Context) (*corev1.Secret, error) {
	logger := log.FromContext(ctx)

	secretList := &corev1.SecretList{}
	labelSelector := client.MatchingLabels{"mintmaker.appstudio.redhat.com/secret-type": "registry"}
	listOptions := []client.ListOption{
		client.InNamespace(MintMakerNamespaceName),
		labelSelector,
	}

	err := r.Client.List(ctx, secretList, listOptions...)

	if err != nil {
		return nil, err
	}

	if len(secretList.Items) == 0 {
		// No secrets to merge
		return nil, nil
	}

	logger.Info(fmt.Sprintf("Found %d secrets to merge", len(secretList.Items)))

	mergedAuths := make(map[string]interface{})
	for _, secret := range secretList.Items {
		if secret.Type == corev1.SecretTypeDockerConfigJson {
			data, exists := secret.Data[".dockerconfigjson"]
			if !exists {
				// No .dockerconfigjson section
				logger.Info("Found secret without .dockerconfigjson section")
				return nil, nil
			}

			var dockerConfig map[string]interface{}
			if err := json.Unmarshal(data, &dockerConfig); err != nil {
				return nil, err
			}

			auths, exists := dockerConfig["auths"].(map[string]interface{})
			if !exists {
				continue
			}

			for registry, creds := range auths {
				mergedAuths[registry] = creds
			}
		}
	}

	mergedDockerConfig := map[string]interface{}{
		"auths": mergedAuths,
	}

	if len(mergedAuths) == 0 {
		logger.Info("Merged auths empty, skipping creation of secret")

		return nil, nil
	}

	mergedConfigJson, err := json.Marshal(mergedDockerConfig)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Unix()
	name := fmt.Sprintf("renovate-image-pull-secrets-%d-%s", timestamp, RandomString(5))

	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: MintMakerNamespaceName,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": []byte(mergedConfigJson),
		},
	}

	if err := r.Client.Create(ctx, newSecret); err != nil {
		return nil, err
	}

	return newSecret, nil
}

// createPipelineRun creates and returns a new PipelineRun
func (r *DependencyUpdateCheckReconciler) createPipelineRun(ctx context.Context, resources *loader.ProcessingResources) (*tektonv1beta1.PipelineRun, error) {

	logger := log.FromContext(ctx)
	timestamp := time.Now().Unix()
	name := fmt.Sprintf("renovate-pipelinerun-%d-%s", timestamp, RandomString(5))

	renovateImageUrl := os.Getenv(RenovateImageEnvName)
	if renovateImageUrl == "" {
		renovateImageUrl = DefaultRenovateImageUrl
	}

	// Create a merged secret for private registries
	registry_secret, err := r.createMergedPullSecret(ctx)
	if err != nil {
		return nil, err
	}

	var renovateCmd []string
	// TODO: this needs to be changed to define the RENOVATE_TOKEN once we will get the secret for the component
	for _, task := range tasks {
		taskId := RandomString(5)
		secretTokens[taskId] = task.Token

		config, err := task.GetPipelineRunConfig(ctx, r.Client, registry_secret)
		if err != nil {
			return err
		}
		configMapData[fmt.Sprintf("%s.json", taskId)] = config
		renovateCmd = append(renovateCmd,
			fmt.Sprintf("RENOVATE_CONFIG_FILE=/configs/%s.json renovate || true", taskId),
		)
	}
	if len(renovateCmd) == 0 {
		return nil
	}

	// Creating the pipelineRun definition
	pipelineRun := &tektonv1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: MintMakerNamespaceName,
		},
		Spec: tektonv1beta1.PipelineRunSpec{
			Status: tektonv1beta1.PipelineRunSpecStatusPending,
			PipelineSpec: &tektonv1beta1.PipelineSpec{
				Tasks: []tektonv1beta1.PipelineTask{
					{
						Name: "build",
						TaskSpec: &tektonv1beta1.EmbeddedTask{
							TaskSpec: tektonv1beta1.TaskSpec{
								Steps: []tektonv1beta1.Step{
									{
										Name:  "renovate",
										Image: renovateImageUrl,
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

	r.Create(ctx, pipelineRun)
	if err := r.Client.Create(ctx, pipelineRun); err != nil {
		return nil, err
	}

	logger.Info(fmt.Sprintf("Created pipelinerun %v", pipelineRun))
	return pipelineRun, nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *DependencyUpdateCheckReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := ctrllog.FromContext(ctx).WithName("DependencyUpdateCheckController")
	ctx = ctrllog.IntoContext(ctx, log)

	// Ignore CRs that are not from the mintmaker namespace
	if req.Namespace != MintMakerNamespaceName {
		return ctrl.Result{}, nil
	}

	dependencyupdatecheck := &mmv1alpha1.DependencyUpdateCheck{}
	err := r.Client.Get(ctx, req.NamespacedName, dependencyupdatecheck)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
	}

	// If the DependencyUpdateCheck has been handled before, skip it
	if value, exists := dependencyupdatecheck.Annotations[MintMakerProcessedAnnotationName]; exists && value == "true" {
		log.Info(fmt.Sprintf("DependencyUpdateCheck has been processed: %v", req.NamespacedName))
		return ctrl.Result{}, nil
	}

	// Update the DependencyUpdateCheck to add a processed annotation
	log.Info(fmt.Sprintf("new DependencyUpdateCheck found: %v", req.NamespacedName))
	if dependencyupdatecheck.Annotations == nil {
		dependencyupdatecheck.Annotations = map[string]string{}
	}
	dependencyupdatecheck.Annotations[MintMakerProcessedAnnotationName] = "true"

	err = r.Client.Update(ctx, dependencyupdatecheck)
	if err != nil {
		log.Error(err, "failed to update DependencyUpdateCheck annotations")
		return ctrl.Result{}, nil
	}

	var gatheredComponents []appstudiov1alpha1.Component
	if len(dependencyupdatecheck.Spec.Workspaces) > 0 {
		log.Info(fmt.Sprintf("Following components are specified: %v", dependencyupdatecheck.Spec.Workspaces))
		gatheredComponents, err = getFilteredComponents(dependencyupdatecheck.Spec.Workspaces, r.Client, ctx)
		if err != nil {
			log.Error(err, "gathering filtered components has failed")
			return ctrl.Result{}, err
		}
	} else {
		allComponents := &appstudiov1alpha1.ComponentList{}
		if err := r.Client.List(ctx, allComponents, &client.ListOptions{}); err != nil {
			log.Error(err, "failed to list Components")
			return ctrl.Result{}, err
		}
		gatheredComponents = allComponents.Items

	}

	log.Info(fmt.Sprintf("%v components will be processed", len(gatheredComponents)))

	// Filter out components which have mintmaker disabled
	componentList := []appstudiov1alpha1.Component{}
	for _, component := range gatheredComponents {
		if value, exists := component.Annotations[MintMakerDisabledAnnotationName]; !exists || value != "true" {
			componentList = append(componentList, component)
		}
	}

	log.Info("found components with mintmaker disabled", "components", len(gatheredComponents)-len(componentList))
	if len(componentList) == 0 {
		return ctrl.Result{}, nil
	}

	var scmComponents []*git.ScmComponent
	for _, component := range componentList {
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

	if len(tasks) == 0 {
		return ctrl.Result{}, nil
	}

	// TODO: we used to start the job here, we need to create a pipeline run in pending state now...

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DependencyUpdateCheckReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// we are monitoring the creation of DependencyUpdateCheck
	return ctrl.NewControllerManagedBy(mgr).
		For(&mmv1alpha1.DependencyUpdateCheck{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc:  func(createEvent event.CreateEvent) bool { return true },
			DeleteFunc:  func(deleteEvent event.DeleteEvent) bool { return false },
			UpdateFunc:  func(updateEvent event.UpdateEvent) bool { return false },
			GenericFunc: func(genericEvent event.GenericEvent) bool { return false },
		}).
		Complete(r)
}
