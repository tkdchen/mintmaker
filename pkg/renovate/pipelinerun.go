package renovate

import (
	"os"

	. "github.com/konflux-ci/mintmaker/pkg/common"
	"github.com/konflux-ci/release-service/loader"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PipelineRunCoordinator is responsible for creating and managing renovate pipelineruns
type PipelinerunCoordinator struct {
	renovateImageUrl string
	debug            bool
	client           client.Client
	scheme           *runtime.Scheme
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
