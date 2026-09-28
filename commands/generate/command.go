package generate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	appcommands "github.com/mantyr/app/commands"
	"github.com/urfave/cli/v2"

	"crd.tools/commands"
	"crd.tools/commands/selection"
	"crd.tools/components/workspace"
	"crd.tools/di"
	"crd.tools/generate"
)

// Флаги переопределения, применяются только к одному выбранному CRD
const (
	outputFlag        = "output"
	setGroupFlag      = "set-group"
	setKindFlag       = "set-kind"
	setPluralFlag     = "set-plural"
	setSingularFlag   = "set-singular"
	setScopeFlag      = "set-scope"
	setShortNamesFlag = "set-short-names"
	setCategoriesFlag = "set-categories"
	setLabelFlag      = "set-label"
	setAnnotationFlag = "set-annotation"
)

// Command это команда generate, собирающая манифесты CRD
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
	c.Command.Name = "generate"
	c.Command.Usage = "собрать манифесты CRD: generate [флаги] [шаблоны пакетов], по умолчанию ."
	c.Command.Description = "Находит пакеты, где crd.Add вызывается в init, собирает определения " +
		"и печатает манифесты или пишет их в каталог -o. С --validate проверяет их кодом API-сервера"
	c.Command.AddFlags(commands.Flags()...)
	c.Command.AddFlags(workspace.Flags(c.settings.Workspace)...)
	c.Command.AddFlags(selection.Flags()...)
	c.Command.AddFlags(
		&cli.StringFlag{Name: outputFlag, Aliases: []string{"o"}, Usage: "каталог для файлов <group>_<plural>.yaml, без него — вывод в stdout"},
		&cli.StringFlag{Name: setGroupFlag, Usage: "заменить group, только для одного CRD"},
		&cli.StringFlag{Name: setKindFlag, Usage: "заменить kind, только для одного CRD"},
		&cli.StringFlag{Name: setPluralFlag, Usage: "заменить plural, только для одного CRD"},
		&cli.StringFlag{Name: setSingularFlag, Usage: "заменить singular, только для одного CRD"},
		&cli.StringFlag{Name: setScopeFlag, Usage: "заменить scope: Namespaced или Cluster, только для одного CRD"},
		&cli.StringSliceFlag{Name: setShortNamesFlag, Usage: "заменить shortNames, только для одного CRD"},
		&cli.StringSliceFlag{Name: setCategoriesFlag, Usage: "заменить categories, только для одного CRD"},
		&cli.StringSliceFlag{Name: setLabelFlag, Usage: "добавить метку key=value, только для одного CRD"},
		&cli.StringSliceFlag{Name: setAnnotationFlag, Usage: "добавить аннотацию key=value, только для одного CRD"},
	)
	return nil
}

// Action собирает манифесты
func (c *Command) Action(ctx *cli.Context) error {
	overrides, err := parseOverrides(ctx)
	if err != nil {
		return cli.Exit(err.Error(), 2)
	}
	container, err := di.New(c.settings.Name)
	if err != nil {
		return err
	}
	patterns, err := selection.Patterns(ctx, "generate")
	if err != nil {
		return err
	}
	return commands.Run(ctx, func(ctx *cli.Context, parent context.Context) error {
		ws := container.Workspace
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
		report, err := engine.Run(parent, generate.Options{
			Dir:       cwd,
			Patterns:  patterns,
			Filter:    selection.Filter(ctx),
			Overrides: overrides,
			Output:    ctx.String(outputFlag),
			Validate:  ctx.Bool(selection.ValidateFlag),
			Stdout:    ctx.App.Writer,
			Stderr:    ctx.App.ErrWriter,
		})
		if err != nil {
			return err
		}
		if len(report.Files) > 0 {
			fmt.Fprintf(ctx.App.ErrWriter, "%s generate: %d CRD from %d packages\n", c.settings.Name, report.CRDs, len(report.Packages))
			for _, f := range report.Files {
				fmt.Fprintf(ctx.App.ErrWriter, "  %s\n", f)
			}
		}
		return nil
	}, container.Workspace)
}

func parseOverrides(ctx *cli.Context) (generate.Overrides, error) {
	o := generate.Overrides{
		Group:      ctx.String(setGroupFlag),
		Kind:       ctx.String(setKindFlag),
		Plural:     ctx.String(setPluralFlag),
		Singular:   ctx.String(setSingularFlag),
		Scope:      ctx.String(setScopeFlag),
		ShortNames: ctx.StringSlice(setShortNamesFlag),
		Categories: ctx.StringSlice(setCategoriesFlag),
	}
	var err error
	if o.Labels, err = keyValues(ctx.StringSlice(setLabelFlag)); err != nil {
		return o, fmt.Errorf("--%s: %w", setLabelFlag, err)
	}
	if o.Annotations, err = keyValues(ctx.StringSlice(setAnnotationFlag)); err != nil {
		return o, fmt.Errorf("--%s: %w", setAnnotationFlag, err)
	}
	if o.Scope != "" && o.Scope != "Namespaced" && o.Scope != "Cluster" {
		return o, fmt.Errorf("--%s: must be Namespaced or Cluster", setScopeFlag)
	}
	return o, nil
}

func keyValues(items []string) (map[string]string, error) {
	if len(items) == 0 {
		return nil, nil
	}
	out := map[string]string{}
	for _, item := range items {
		k, v, ok := strings.Cut(item, "=")
		if !ok || k == "" {
			return nil, errors.New("expected key=value, got " + item)
		}
		out[k] = v
	}
	return out, nil
}
