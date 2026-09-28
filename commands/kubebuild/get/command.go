package get

import (
	"context"
	"fmt"
	"os"

	appcommands "github.com/mantyr/app/commands"
	"github.com/urfave/cli/v2"

	"crd.tools/commands"
	"crd.tools/components/kubebuild"
	"crd.tools/components/workspace"
	"crd.tools/di"
)

// Command это команда kubebuild get
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
	c.Command.Name = "get"
	c.Command.Usage = "добавить схемы для меток в пакетах: get [шаблоны пакетов], по умолчанию ."
	c.Command.Description = fmt.Sprintf(
		"Ищет поля с меткой %s в пакетах по шаблонам (., ./..., путь импорта) и дособирает схемы. "+
			"Ничего не удаляет. Внешние пакеты запоминаются в настройках для prune",
		c.settings.Kubebuild.Tag,
	)
	c.Command.AddFlags(commands.Flags()...)
	c.Command.AddFlags(workspace.Flags(c.settings.Workspace)...)
	c.Command.AddFlags(kubebuild.Flags(c.settings.Kubebuild)...)
	return nil
}

// Action собирает схемы
func (c *Command) Action(ctx *cli.Context) error {
	container, err := di.New(c.settings.Name)
	if err != nil {
		return err
	}
	patterns := ctx.Args().Slice()
	return commands.Run(ctx, func(ctx *cli.Context, parent context.Context) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		report, err := container.Kubebuild.Get(parent, cwd, patterns)
		if err != nil {
			return err
		}
		report.Print(ctx.App.Writer, c.settings.Name+" kubebuild get")
		return nil
	}, container.Workspace, container.Kubebuild)
}
