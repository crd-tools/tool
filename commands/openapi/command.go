package openapi

import (
	"path/filepath"

	app "github.com/mantyr/app"
	appcommands "github.com/mantyr/app/commands"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"

	"crd.tools/commands"
)

// ConfigFlag флаг файла настроек OpenAPI
const ConfigFlag = "openapi.config"

// Command это родительская команда openapi
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
	c.Command.Name = "openapi"
	c.Command.Usage = "документ OpenAPI v3 по определениям CRD, каждый тип — отдельный компонент"
	return nil
}

// Subcommands возвращает подкоманды
func (c *Command) Subcommands() []app.Command {
	return []app.Command{
		newGenerate(c.settings),
		newPrefix(c.settings),
	}
}

// configFlag возвращает флаг файла настроек
func configFlag(s commands.Settings) cli.Flag {
	return altsrc.NewStringFlag(&cli.StringFlag{
		Name:    ConfigFlag,
		Usage:   "файл настроек OpenAPI относительно корня модуля",
		EnvVars: []string{s.Env + "_OPENAPI_CONFIG"},
		Value:   s.OpenAPIFile,
	})
}

// configPath возвращает путь к файлу настроек
func configPath(ctx *cli.Context, root string) string {
	file := ctx.String(ConfigFlag)
	if filepath.IsAbs(file) {
		return file
	}
	return filepath.Join(root, file)
}
