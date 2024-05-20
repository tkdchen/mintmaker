package renovate

import (
	"k8s.io/utils/strings/slices"
)

const (
	RenovateConfigMapName = "renovate-config"
	RenovateConfigKey     = "renovate.json"
)

type Repository struct {
	Repository   string   `json:"repository"`
	BaseBranches []string `json:"baseBranches"`
}

func (r *Repository) AddBranch(branch string) {
	if !slices.Contains(r.BaseBranches, branch) {
		r.BaseBranches = append(r.BaseBranches, branch)
	}
}
