package common

const (
	MintMakerNamespaceName = "mintmaker"
	//
	MintMakerProcessedAnnotationName = "mintmaker.appstudio.redhat.com/processed"
	// Pipelines as Code GitHub appliaction configuration secret name.
	// The secret is located in Build Service namespace.
	PipelinesAsCodeGitHubAppSecretName = "pipelines-as-code-secret"
	// Keys of the GitHub app ID and the app private key in the pipelines-as-code-secret
	PipelinesAsCodeGithubAppIdKey   = "github-application-id"
	PipelinesAsCodeGithubPrivateKey = "github-private-key"
)
