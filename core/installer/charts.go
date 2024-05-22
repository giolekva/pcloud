package installer

import (
	"strings"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
)

type LocalChartGenerator interface {
	Generate(path string) helmv2.HelmChartTemplateSpec
}

type GitRepositoryLocalChartGenerator struct {
	Name      string
	Namespace string
}

func (g GitRepositoryLocalChartGenerator) Generate(path string) helmv2.HelmChartTemplateSpec {
	p, _ := strings.CutPrefix(path, "/")
	return helmv2.HelmChartTemplateSpec{
		Chart: p,
		SourceRef: helmv2.CrossNamespaceObjectReference{
			Kind:      "GitRepository",
			Name:      g.Name,
			Namespace: g.Namespace,
		},
	}
}

type InfraLocalChartGenerator struct {
	GitRepositoryLocalChartGenerator
}

func NewInfraLocalChartGenerator() InfraLocalChartGenerator {
	return InfraLocalChartGenerator{GitRepositoryLocalChartGenerator{"dodo-flux", "dodo-flux"}}
}
