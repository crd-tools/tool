package kubebuild

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v2"

	"crd.tools/components/workspace"
	"crd.tools/kubebuild"
)

// Component это компонент сбора схем из kubebuilder-маркеров
type Component struct {
	dep       Dep
	config    Config
	workspace workspace.Workspace
	engine    *kubebuild.Engine
}

// New возвращает компонент
func New(dep Dep) *Component {
	return &Component{dep: dep}
}

// Name возвращает имя компонента
func (c *Component) Name() string {
	return "Kubebuild"
}

// Init создаёт движок по настройкам проекта
func (c *Component) Init(ctx *cli.Context) error {
	if c.dep == nil {
		return errors.New("empty dep")
	}
	c.workspace = c.dep.GetWorkspace()
	if c.workspace == nil {
		return errors.New("empty workspace in dep")
	}
	tool := c.dep.GetToolName()
	c.config = Parse(ctx)

	p, err := c.workspace.Project()
	if errors.Is(err, workspace.ErrNotInitialized) {
		return fmt.Errorf("%w, run: %s init", err, tool)
	}
	if err != nil {
		return err
	}
	c.engine, err = kubebuild.New(kubebuild.Settings{
		Tool:       tool,
		Root:       c.workspace.Root(),
		ModulePath: c.workspace.ModulePath(),
		Output:     p.Output,
		Tag:        c.config.Tag,
		Generator:  kubebuild.GeneratorVersion(),
	})
	return err
}

// Destroy ничего не освобождает
func (c *Component) Destroy(ctx *cli.Context) error {
	return nil
}

// Get добавляет схемы для меток в пакетах по шаблонам
// Внешние шаблоны запоминаются в настройках проекта, чтобы их учитывал prune
func (c *Component) Get(ctx context.Context, dir string, patterns []string) (kubebuild.Report, error) {
	report, err := c.engine.Get(ctx, dir, patterns)
	if err != nil {
		return report, err
	}
	p, err := c.workspace.Project()
	if err != nil {
		return report, err
	}
	if p.AddExternal(report.External...) {
		if err := c.workspace.Save(p); err != nil {
			return report, err
		}
	}
	return report, nil
}

// Prune пересобирает схемы по всем пакетам проекта и удаляет неиспользуемые
func (c *Component) Prune(ctx context.Context) (kubebuild.Report, error) {
	p, err := c.workspace.Project()
	if err != nil {
		return kubebuild.Report{}, err
	}
	return c.engine.Prune(ctx, p.Kubebuild.Scan, p.Kubebuild.External)
}
