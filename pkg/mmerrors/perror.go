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

package mmerrors

import (
	"errors"
	"fmt"
)

var _ error = (*MintMakerError)(nil)

// MintMakerError extends standard error to:
//  1. Keep persistent / transient property of the error.
//     All errors, except ETransientErrorId considered persistent.
//  2. Have error ID to show the root cause of the error and optionally short message.
type MintMakerError struct {
	// id is used to determine if error is persistent and to know the root cause of the error
	id MMErrorId
	// typically used to log the error message along with nested errors
	err error
	// Optional. To provide extra information about this error
	// If set, it will be appended to the error message returned from Error
	ExtraInfo string
}

func NewMintMakerError(id MMErrorId, err error) *MintMakerError {
	return &MintMakerError{
		id:        id,
		err:       err,
		ExtraInfo: "",
	}
}

func (r MintMakerError) Error() string {
	if r.err == nil {
		return r.ShortError()
	}
	if r.ExtraInfo == "" {
		return r.err.Error()
	}
	return fmt.Sprintf("%s %s", r.err.Error(), r.ExtraInfo)
}

func (r MintMakerError) GetErrorId() int {
	return int(r.id)
}

// ShortError returns short message with error ID in case of persistent error or
// standard error message for transient errors.
func (r MintMakerError) ShortError() string {
	if r.id == ETransientError {
		if r.err != nil {
			return r.err.Error()
		}
		return "transient error"
	}
	return fmt.Sprintf("%d: %s", r.id, mmErrorMessages[r.id])
}

func (r MintMakerError) IsPersistent() bool {
	return r.id != ETransientError
}

type MMErrorId int

const (
	ETransientError MMErrorId = 0
	EUnknownError   MMErrorId = 1

	// 'pipelines-as-code-secret' secret doesn't exists in 'mintmaker' namespace nor Component's one.
	EPaCSecretNotFound MMErrorId = 50
	// Validation of 'pipelines-as-code-secret' secret failed
	EPaCSecretInvalid MMErrorId = 51
	// Pipelines as Code public route to recieve webhook events doesn't exist in expected namespaces.
	EPaCRouteDoesNotExist MMErrorId = 52
	// An attempt to create another PaC repository object that references the same git repository.
	EPaCDuplicateRepository MMErrorId = 53
	// Git repository url isn't allowed
	EPaCNotAllowedRepositoryUrl MMErrorId = 54

	// Happens when Component source repository is hosted on unsupported / unknown git provider.
	// For example: https://my-gitlab.com
	// If self-hosted instance of the supported git providers is used, then "git-provider" annotation must be set:
	// git-provider: gitlab
	EUnknownGitProvider MMErrorId = 60
	// Insecure HTTP can't be used for git repository URL
	EHttpUsedForRepository MMErrorId = 61

	// Happens when configured in cluster Pipelines as Code application is not installed in Component source repository.
	// User must install the application to fix this error.
	EGitHubAppNotInstalled MMErrorId = 70
	// Bad formatted private key
	EGitHubAppMalformedPrivateKey MMErrorId = 71
	// GitHub Application ID is not a valid integer
	EGitHubAppMalformedId MMErrorId = 77
	// Private key doesn't match the GitHub Application
	EGitHubAppPrivateKeyNotMatched MMErrorId = 72
	// GitHub Application with specified ID does not exists.
	// Correct configuration in the AppStudio installation ('pipelines-as-code-secret' secret in 'mintmaker' namespace).
	EGitHubAppDoesNotExist MMErrorId = 73
	// EGitHubAppSuspended Application in git repository is suspended
	EGitHubAppSuspended MMErrorId = 78

	// EGitHubTokenUnauthorized access token can't be recognized by GitHub and 401 is responded.
	// This error may be caused by a malformed token string or an expired token.
	EGitHubTokenUnauthorized MMErrorId = 74
	// EGitHubNoResourceToOperateOn No resource is suitable for GitHub to handle the request and 404 is responded.
	// Generally, this error could be caused by two cases. One is, operate non-existing resource with an access
	// token that has sufficient scope, e.g. delete a non-existing webhook. Another one is, the access token does
	// not have sufficient scope, e.g. list webhooks from a repository, but scope "read:repo_hook" is set.
	EGitHubNoResourceToOperateOn MMErrorId = 75
	// EGitHubReachRateLimit reach the GitHub REST API rate limit.
	EGitHubReachRateLimit MMErrorId = 76
	// EGitHubSecretInvalid the secret with GitHub App credentials is invalid.
	EGitHubSecretInvalid = 77
	// EGitHubSecretTypeNotSupported the secret type with GitHub App credentials is not supported.
	EGitHubSecretTypeNotSupported = 78

	// EGitLabTokenUnauthorized access token is not recognized by GitLab and 401 is responded.
	// The access token may be malformed or expired.
	EGitLabTokenUnauthorized MMErrorId = 90
	// EGitLabTokenInsufficientScope the access token does not have sufficient scope and 403 is responded.
	EGitLabTokenInsufficientScope MMErrorId = 91
	// EGitLabSecretInvalid the secret with GitLab credentials is invalid.
	EGitLabSecretInvalid MMErrorId = 92
	// EGitLabSecretTypeNotSupported the secret type with GitLab credentials is not supported.
	EGitLabSecretTypeNotSupported MMErrorId = 93

	// Value of 'image.redhat.com/image' component annotation is not a valid json or the json has invalid structure.
	EFailedToParseImageAnnotation MMErrorId = 200
	// The secret with git credentials specified in component.Spec.Secret does not exist in the user's namespace.
	EComponentGitSecretMissing MMErrorId = 201
	// The secret with image registry credentials specified in 'image.redhat.com/image' annotation does not exist in the user's namespace.
	EComponentImageRegistrySecretMissing MMErrorId = 202
	// The secret with git credentials not given for component with private git repository.
	EComponentGitSecretNotSpecified MMErrorId = 203

	// EInvalidDevfile devfile of the component is not valid.
	EInvalidDevfile MMErrorId = 220

	// ENoPipelineIsSelected no pipeline can be selected based on a component repository
	ENoPipelineIsSelected MMErrorId = 300
	// EBuildPipelineSelectorNotDefined A BuildPipelineSelector CR cannot be found from all supported search places and with supported names.
	EBuildPipelineSelectorNotDefined MMErrorId = 301
	// EUnsupportedPipelineRef The pipelineRef selected for a component (based on a BuildPipelineSelector)
	// uses a feature that mintmaker does not support (e.g. unsupported resolver).
	EUnsupportedPipelineRef MMErrorId = 302
	// EMissingParamsForBundleResolver The pipelineRef selected for a component is missing parameters required for the bundle resolver.
	EMissingParamsForBundleResolver MMErrorId = 303

	// EPipelineRetrievalFailed Failed to retrieve a Tekton Pipeline.
	EPipelineRetrievalFailed MMErrorId = 400
	// EPipelineConversionFailed Failed to convert a Tekton Pipeline to the version
	// that mintmaker supports (e.g. tekton.dev/v1beta1 -> tekton.dev/v1).
	EPipelineConversionFailed MMErrorId = 401
)

var mmErrorMessages = map[MMErrorId]string{
	ETransientError: "",
	EUnknownError:   "unknown error",

	EPaCSecretNotFound:          "Pipelines as Code secret does not exist",
	EPaCSecretInvalid:           "Invalid Pipelines as Code secret",
	EPaCRouteDoesNotExist:       "Pipelines as Code public route does not exist",
	EPaCDuplicateRepository:     "Git repository is already handled by Pipelines as Code",
	EPaCNotAllowedRepositoryUrl: "Git repository url isn't allowed",

	EUnknownGitProvider:    "unknown git provider of the source repository",
	EHttpUsedForRepository: "http used for git repository, use secure connection",

	EGitHubAppNotInstalled:         "GitHub Application is not installed in user repository",
	EGitHubAppMalformedPrivateKey:  "malformed GitHub Application private key",
	EGitHubAppMalformedId:          "malformed GitHub Application ID",
	EGitHubAppPrivateKeyNotMatched: "GitHub Application private key does not match Application ID",
	EGitHubAppDoesNotExist:         "GitHub Application with given ID does not exist",
	EGitHubAppSuspended:            "GitHub Application is suspended for repository",

	EGitHubTokenUnauthorized:     "Access token is unrecognizable by GitHub",
	EGitHubNoResourceToOperateOn: "No resource for finishing the request",
	EGitHubReachRateLimit:        "Reach GitHub REST API rate limit",

	EGitLabTokenInsufficientScope: "GitLab access token does not have enough scope",
	EGitLabTokenUnauthorized:      "Access token is unrecognizable by remote GitLab service",

	EFailedToParseImageAnnotation:        "Failed to parse image.redhat.com/image annotation value",
	EComponentGitSecretMissing:           "Secret with git credential not found",
	EComponentImageRegistrySecretMissing: "Component image repository secret not found",
	EComponentGitSecretNotSpecified:      "Git credentials for private Component git repository not given",

	EInvalidDevfile: "Component Devfile is invalid",

	ENoPipelineIsSelected:            "No pipeline is selected for component repository based on predefined selectors.",
	EBuildPipelineSelectorNotDefined: "Build pipeline selector is not defined yet.",
	EUnsupportedPipelineRef:          "The pipelineRef for this component (based on pipeline selectors) is not supported.",
	EMissingParamsForBundleResolver:  "The pipelineRef for this component is missing required parameters ('name' and/or 'bundle').",

	EPipelineRetrievalFailed:  "Failed to retrieve the pipeline selected for this component.",
	EPipelineConversionFailed: "Failed to convert the selected pipeline to the supported Tekton API version.",
}

// IsMintMakerError returns true if the specified error is MintMakerError with certain code.
func IsMintMakerError(err error, code MMErrorId) bool {
	var mmErr *MintMakerError
	if err != nil && errors.As(err, &mmErr) {
		if mmErr.GetErrorId() == int(code) {
			return true
		}
	}
	return false
}
