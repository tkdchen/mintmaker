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

package gitprovider

import (
	"net/url"
	"os"
	"strings"
)

const (
	PipelinesAsCodeWebhhokInsecureSslEnvVar = "PAC_WEBHOOK_INSECURE_SSL"
)

func IsInsecureSSL() bool {
	if insecureSSLVal := os.Getenv(PipelinesAsCodeWebhhokInsecureSslEnvVar); insecureSSLVal != "" {
		disableValues := []string{"1", "true", "True"}
		for _, val := range disableValues {
			if insecureSSLVal == val {
				return true
			}
		}
	}
	return false
}

func ParseGitURL(gitUrl string) (*url.URL, error) {
	gitUrl = strings.TrimSuffix(strings.TrimSuffix(gitUrl, ".git"), "/")

	if strings.HasPrefix(gitUrl, "git@") {
		gitUrl = strings.Replace(gitUrl, ":", "/", 1)
		gitUrl = strings.Replace(gitUrl, "git@", "https://", 1)
	}

	parsedUrl, err := url.Parse(gitUrl)
	if err != nil {
		return nil, err
	}

	return parsedUrl, nil
}
