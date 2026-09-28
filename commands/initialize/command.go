package initialize

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	appcommands "github.com/mantyr/app/commands"
	"github.com/urfave/cli/v2"

	"crd.tools/commands"
	"crd.tools/components/workspace"
	"crd.tools/di"
	"crd.tools/project"
)

// Command это команда init, создающая файл настроек проекта
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
	c.Command.Name = "init"
	c.Command.Usage = "создать файл настроек проекта: init [каталог пакета со схемами]"
	c.Command.Description = fmt.Sprintf(
		"Создаёт %s в корне модуля. Каталог пакета со схемами по умолчанию: %s",
		c.settings.Workspace.ConfigFile, c.settings.Output,
	)
	c.Command.AddFlags(commands.Flags()...)
	c.Command.AddFlags(workspace.Flags(c.settings.Workspace)...)
	return nil
}

// Action создаёт файл настроек
func (c *Command) Action(ctx *cli.Context) error {
	if ctx.NArg() > 1 {
		return cli.Exit("init accepts at most one argument", 2)
	}
	container, err := di.New(c.settings.Name)
	if err != nil {
		return err
	}
	return commands.Run(ctx, func(ctx *cli.Context, parent context.Context) error {
		return c.run(ctx, container.Workspace)
	}, container.Workspace)
}

func (c *Command) run(ctx *cli.Context, ws workspace.Workspace) error {
	if ws.Exists() {
		return fmt.Errorf("already initialized: %s", ws.ConfigFile())
	}
	output := c.settings.Output
	if ctx.NArg() == 1 {
		var err error
		output, err = outputPath(ws.Root(), ctx.Args().First())
		if err != nil {
			return err
		}
	}
	if err := ws.Save(project.New(output)); err != nil {
		return err
	}
	fmt.Fprintf(ctx.App.Writer, "created %s\n  output: %s\n", ws.ConfigFile(), output)
	fmt.Fprintf(ctx.App.Writer, "next: %s kubebuild get ./...\n", c.settings.Name)
	return nil
}

// outputPath приводит путь из аргумента к пути относительно корня модуля
func outputPath(root, arg string) (string, error) {
	abs, err := filepath.Abs(arg)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", err
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("output must be a directory inside the module: %s", arg)
	}
	if st, err := os.Stat(abs); err == nil && !st.IsDir() {
		return "", fmt.Errorf("output is not a directory: %s", arg)
	}
	return rel, nil
}
