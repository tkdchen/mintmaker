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

// What to test:
// - check countRunningPipelineRuns
// - with a queue of 3 pipelineruns, when one finishes one other starts and one stays pending

package controller

import (
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	local_tekton "github.com/konflux-ci/mintmaker/internal/pkg/tekton"
	. "github.com/konflux-ci/mintmaker/pkg/common"
)

var _ = Describe("PipelineRun Controller", func() {
	Context("When reconciling pipelineruns", func() {
		_ = BeforeEach(func() {
			createNamespace(MintMakerNamespaceName)

			for i := range 3 {
				pplrName := "pplnr" + strconv.Itoa(i)
				pipelineRunBuilder := local_tekton.NewPipelineRunBuilder(pplrName, MintMakerNamespaceName)
				pipelinerun, err := pipelineRunBuilder.Build()
				Expect(err).NotTo(HaveOccurred())
				Expect(k8sClient.Create(ctx, pipelinerun)).Should(Succeed())
			}
			Eventually(listPipelineRuns).WithArguments(MintMakerNamespaceName).Should(HaveLen(3))
		})

		_ = AfterEach(func() {
			pipelineRuns := listPipelineRuns(MintMakerNamespaceName)
			for _, pipelinerun := range pipelineRuns {
				Expect(k8sClient.Delete(ctx, &pipelinerun)).Should(Succeed())
			}
		})

		It("should ensure pending status is removed from all PipelineRuns", func() {
			Eventually(func() bool {
				pipelineRuns := listPipelineRuns(MintMakerNamespaceName)
				for _, plr := range pipelineRuns {
					if plr.Spec.Status != "" {
						return false
					}
				}
				return true
			}, time.Second*5, time.Millisecond*100).Should(BeTrue())
		})
	})
})
