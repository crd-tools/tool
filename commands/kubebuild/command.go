package kubebuild

import (
	app "github.com/mantyr/app"
	appcommands "github.com/mantyr/app/commands"

	"crd.tools/commands"
	"crd.tools/commands/kubebuild/get"
	"crd.tools/commands/kubebuild/prune"
)

// Command это родительская команда kubebuild
type Command struct {
	appcommands.Command
	settings commands.Settings
}

// New возвращает команду
func New(settings commands.Settings) *Command {
	return &Command{settings: settings}
}

// Init описывает команду
func (c *Command) Init() error {
	c.Command.Name = "kubebuild"
	c.Command.Usage = "схемы типов с kubebuilder-маркерами"
	return nil
}

// Subcommands возвращает подкоманды
func (c *Command) Subcommands() []app.Command {
	return []app.Command{
		get.New(c.settings),
		prune.New(c.settings),
	}
}
