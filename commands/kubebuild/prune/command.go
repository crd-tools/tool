package prune

import (
	"context"

	appcommands "github.com/mantyr/app/commands"
	"github.com/urfave/cli/v2"

	"crd.tools/commands"
	"crd.tools/components/kubebuild"
	"crd.tools/components/workspace"
	"crd.tools/di"
)

// Command это команда kubebuild prune
type Command struct {
	appcommands.Command
	settings commands.Settings
}

// New возвращает команду
func New(settings commands.Settings) *Command {
	return &Command{settings: settings}
}

// Init описывает команду и её флаги
func (c *Command) Init() error {
	c.Command.Name = "prune"
	c.Command.Usage = "пересобрать схемы по всему проекту и удалить неиспользуемые"
	c.Command.Description = "Обходит шаблоны kubebuild.scan и kubebuild.external из настроек проекта " +
		"независимо от текущего каталога"
	c.Command.AddFlags(commands.Flags()...)
	c.Command.AddFlags(workspace.Flags(c.settings.Workspace)...)
	c.Command.AddFlags(kubebuild.Flags(c.settings.Kubebuild)...)
	return nil
}

// Action пересобирает схемы
func (c *Command) Action(ctx *cli.Context) error {
	if ctx.NArg() > 0 {
		return cli.Exit("prune takes no arguments, patterns are read from the project config", 2)
	}
	container, err := di.New(c.settings.Name)
	if err != nil {
		return err
	}
	return commands.Run(ctx, func(ctx *cli.Context, parent context.Context) error {
		report, err := container.Kubebuild.Prune(parent)
		if err != nil {
			return err
		}
		report.Print(ctx.App.Writer, c.settings.Name+" kubebuild prune")
		return nil
	}, container.Workspace, container.Kubebuild)
}
