package openapi

import (
	"context"
	"fmt"
	"text/tabwriter"

	app "github.com/mantyr/app"
	appcommands "github.com/mantyr/app/commands"
	"github.com/urfave/cli/v2"

	"crd.tools/commands"
	"crd.tools/components/workspace"
	"crd.tools/di"
	"crd.tools/openapiconfig"
)

// prefixCommand это родительская команда openapi prefix
type prefixCommand struct {
	appcommands.Command
	settings commands.Settings
}

func newPrefix(settings commands.Settings) *prefixCommand {
	return &prefixCommand{settings: settings}
}

// Init описывает команду
func (c *prefixCommand) Init() error {
	c.Command.Name = "prefix"
	c.Command.Usage = "замены начала имён компонентов: k8s.io.apimachinery.pkg → apimachinery"
	return nil
}

// Subcommands возвращает подкоманды
func (c *prefixCommand) Subcommands() []app.Command {
	return []app.Command{
		&prefixAction{settings: c.settings, name: "add", usage: "добавить или изменить замену: prefix add <from> [to], без to начало отрезается", run: add},
		&prefixAction{settings: c.settings, name: "remove", usage: "удалить замену: prefix remove <from>", run: remove},
		&prefixAction{settings: c.settings, name: "list", usage: "показать замены", run: list},
	}
}

// prefixAction это подкоманда, которая правит или показывает файл настроек
type prefixAction struct {
	appcommands.Command
	settings commands.Settings
	name     string
	usage    string
	run      func(ctx *cli.Context, cfg *openapiconfig.Config) (save bool, err error)
}

// Init описывает команду и её флаги
func (c *prefixAction) Init() error {
	c.Command.Name = c.name
	c.Command.Usage = c.usage
	c.Command.AddFlags(commands.Flags()...)
	c.Command.AddFlags(workspace.Flags(c.settings.Workspace)...)
	c.Command.AddFlags(configFlag(c.settings))
	return nil
}

// Action читает файл, выполняет действие и сохраняет изменения
func (c *prefixAction) Action(ctx *cli.Context) error {
	container, err := di.New(c.settings.Name)
	if err != nil {
		return err
	}
	return commands.Run(ctx, func(ctx *cli.Context, parent context.Context) error {
		file := configPath(ctx, container.Workspace.Root())
		cfg, err := openapiconfig.Load(file)
		if err != nil {
			return err
		}
		save, err := c.run(ctx, cfg)
		if err != nil || !save {
			return err
		}
		return cfg.Save(file)
	}, container.Workspace)
}

func add(ctx *cli.Context, cfg *openapiconfig.Config) (bool, error) {
	if ctx.NArg() < 1 || ctx.NArg() > 2 {
		return false, fmt.Errorf("usage: prefix add <from> [to]")
	}
	return true, cfg.Add(ctx.Args().Get(0), ctx.Args().Get(1))
}

func remove(ctx *cli.Context, cfg *openapiconfig.Config) (bool, error) {
	if ctx.NArg() != 1 {
		return false, fmt.Errorf("usage: prefix remove <from>")
	}
	if !cfg.Remove(ctx.Args().First()) {
		return false, fmt.Errorf("prefix %s not found", ctx.Args().First())
	}
	return true, nil
}

func list(ctx *cli.Context, cfg *openapiconfig.Config) (bool, error) {
	tw := tabwriter.NewWriter(ctx.App.Writer, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "FROM\tTO")
	for _, p := range cfg.Prefixes {
		to := p.To
		if to == "" {
			to = "(отрезать)"
		}
		fmt.Fprintf(tw, "%s\t%s\n", p.From, to)
	}
	return false, tw.Flush()
}
