/*
Copyright 2024 Red Hat, Inc.

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
	"fmt"
	"net/url"
	"strings"

	appstudiov1alpha1 "github.com/redhat-appstudio/application-api/api/v1alpha1"
)

const (
	// Annotation that specifies git provider id for self hosted SCM instances, e.g. github or gitlab.
	GitProviderAnnotationName = "git-provider"
)

func getGitProvider(component appstudiov1alpha1.Component) (string, error) {
	allowedGitProviders := map[string]bool{"github": true, "gitlab": true}
	gitProvider := ""

	if component.Spec.Source.GitSource == nil {
		err := fmt.Errorf("git source URL is not set for %s Component in %s namespace", component.Name, component.Namespace)
		return "", err
	}
	sourceUrl := component.Spec.Source.GitSource.URL

	if strings.HasPrefix(sourceUrl, "git@") {
		// git@github.com:redhat-appstudio/application-service.git
		sourceUrl = strings.TrimPrefix(sourceUrl, "git@")
		host := strings.Split(sourceUrl, ":")[0]
		gitProvider = strings.Split(host, ".")[0]
	} else {
		// https://github.com/redhat-appstudio/application-service
		u, err := url.Parse(sourceUrl)
		if err != nil {
			return "", err
		}
		uParts := strings.Split(u.Hostname(), ".")
		if len(uParts) == 1 {
			gitProvider = uParts[0]
		} else {
			gitProvider = uParts[len(uParts)-2]
		}
	}

	var err error
	if !allowedGitProviders[gitProvider] {
		// Self-hosted git provider, check for git-provider annotation on the component
		gitProviderAnnotationValue := component.GetAnnotations()[GitProviderAnnotationName]
		if gitProviderAnnotationValue != "" {
			if allowedGitProviders[gitProviderAnnotationValue] {
				gitProvider = gitProviderAnnotationValue
			} else {
				err = fmt.Errorf("unsupported \"%s\" annotation value: %s", GitProviderAnnotationName, gitProviderAnnotationValue)
			}
		} else {
			err = fmt.Errorf("self-hosted git provider is not specified via \"%s\" annotation in the component", GitProviderAnnotationName)
		}
	}

	return gitProvider, err
}
