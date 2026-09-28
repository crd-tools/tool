package openapi

import (
	"context"
	"os"

	appcommands "github.com/mantyr/app/commands"
	"github.com/urfave/cli/v2"
	"sigs.k8s.io/yaml"

	"crd.tools/commands"
	"crd.tools/commands/selection"
	"crd.tools/components/workspace"
	"crd.tools/di"
	"crd.tools/generate"
	"crd.tools/openapiconfig"
)

// Флаги команды generate
const (
	outputFlag = "output"
	formatFlag = "format"
)

type generateCommand struct {
	appcommands.Command
	settings commands.Settings
}

func newGenerate(settings commands.Settings) *generateCommand {
	return &generateCommand{settings: settings}
}

// Init описывает команду и её флаги
func (c *generateCommand) Init() error {
	c.Command.Name = "generate"
	c.Command.Usage = "построить документ OpenAPI: openapi generate [флаги] [шаблоны пакетов], по умолчанию ."
	c.Command.Description = "Находит определения так же, как crd generate, и строит документ OpenAPI v3, " +
		"где каждый именованный тип — отдельный компонент. Имена компонентов сокращаются заменами " +
		"префиксов из файла настроек"
	c.Command.AddFlags(commands.Flags()...)
	c.Command.AddFlags(workspace.Flags(c.settings.Workspace)...)
	c.Command.AddFlags(selection.FilterFlags()...)
	c.Command.AddFlags(
		configFlag(c.settings),
		&cli.StringFlag{Name: outputFlag, Aliases: []string{"o"}, Usage: "файл документа, без него — вывод в stdout"},
		&cli.StringFlag{Name: formatFlag, Usage: "yaml или json", Value: "yaml"},
	)
	return nil
}

// Action строит документ
func (c *generateCommand) Action(ctx *cli.Context) error {
	patterns, err := selection.Patterns(ctx, "openapi generate")
	if err != nil {
		return err
	}
	format := ctx.String(formatFlag)
	if format != "yaml" && format != "json" {
		return cli.Exit("--format: must be yaml or json", 2)
	}
	container, err := di.New(c.settings.Name)
	if err != nil {
		return err
	}
	return commands.Run(ctx, func(ctx *cli.Context, parent context.Context) error {
		ws := container.Workspace
		cfg, err := openapiconfig.Load(configPath(ctx, ws.Root()))
		if err != nil {
			return err
		}
		engine, err := generate.New(generate.Settings{
			Tool:           c.settings.Name,
			Root:           ws.Root(),
			SchemasPackage: selection.SchemasPackage(ws),
		})
		if err != nil {
			return err
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		settings := generate.OpenAPISettings{Title: cfg.Info.Title, Version: cfg.Info.Version}
		if settings.Title == "" {
			settings.Title = ws.ModulePath()
		}
		if settings.Version == "" {
			settings.Version = "v1"
		}
		for _, p := range cfg.Prefixes {
			settings.Prefixes = append(settings.Prefixes, generate.Prefix{From: p.From, To: p.To})
		}
		doc, err := engine.OpenAPI(parent, generate.Options{
			Dir:      cwd,
			Patterns: patterns,
			Filter:   selection.Filter(ctx),
		}, settings)
		if err != nil {
			return err
		}
		if format == "yaml" {
			if doc, err = yaml.JSONToYAML(doc); err != nil {
				return err
			}
		} else {
			doc = append(doc, '\n')
		}
		if out := ctx.String(outputFlag); out != "" {
			return os.WriteFile(out, doc, 0o644)
		}
		_, err = ctx.App.Writer.Write(doc)
		return err
	}, container.Workspace)
}
