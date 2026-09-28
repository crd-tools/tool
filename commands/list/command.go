package list

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	appcommands "github.com/mantyr/app/commands"
	"github.com/mantyr/formatter"
	"github.com/urfave/cli/v2"

	"crd.tools/commands"
	"crd.tools/commands/selection"
	"crd.tools/components/workspace"
	"crd.tools/di"
	"crd.tools/generate"
)

// formatFlag флаг формата вывода
const formatFlag = "format"

// Command это команда list, показывающая найденные определения CRD
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
	c.Command.Name = "list"
	c.Command.Usage = "показать определения CRD: list [флаги] [шаблоны пакетов], по умолчанию ."
	c.Command.Description = "Находит определения так же, как generate, и печатает их заголовки: " +
		"имя, kind, group, plural, scope, версии (* — версия хранения) и статус"
	c.Command.AddFlags(commands.Flags()...)
	c.Command.AddFlags(workspace.Flags(c.settings.Workspace)...)
	c.Command.AddFlags(selection.Flags()...)
	c.Command.AddFlags(&cli.StringFlag{
		Name: formatFlag,
		Usage: "формат вывода: table, json, yaml, raw, шаблон text/template или table <шаблон>; поля: " +
			".Name .Kind .Group .Plural .Scope .Versions .Storage .Status .Error, метод .VersionsMarked",
		Value: formatter.TableFormatKey,
	})
	return nil
}

// Action печатает список
func (c *Command) Action(ctx *cli.Context) error {
	patterns, err := selection.Patterns(ctx, "list")
	if err != nil {
		return err
	}
	container, err := di.New(c.settings.Name)
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
		rows, err := engine.List(parent, generate.Options{
			Dir:      cwd,
			Patterns: patterns,
			Filter:   selection.Filter(ctx),
			Validate: ctx.Bool(selection.ValidateFlag),
		})
		if err != nil {
			return err
		}
		failed, err := printRows(ctx.App.Writer, ctx.String(formatFlag), rows)
		if err != nil {
			return err
		}
		if failed > 0 {
			return fmt.Errorf("%d of %d CRD with errors", failed, len(rows))
		}
		return nil
	}, container.Workspace)
}

// Item это элемент списка для вывода, все поля строковые, чтобы их понимал формат raw
type Item struct {
	// Name имя CRD: plural.group
	Name string `json:"name" yaml:"name" raw:"column:name"`

	// Kind вид ресурса
	Kind string `json:"kind" yaml:"kind" raw:"column:kind"`

	// Group группа API
	Group string `json:"group" yaml:"group" raw:"column:group"`

	// Plural имя ресурса во множественном числе
	Plural string `json:"plural" yaml:"plural" raw:"column:plural"`

	// Scope область видимости
	Scope string `json:"scope" yaml:"scope" raw:"column:scope"`

	// Versions версии через запятую
	Versions string `json:"versions" yaml:"versions" raw:"column:versions"`

	// Storage версия хранения
	Storage string `json:"storage" yaml:"storage" raw:"column:storage"`

	// Status ok или error
	Status string `json:"status" yaml:"status" raw:"column:status"`

	// Error ошибка сборки или проверки
	Error string `json:"error,omitempty" yaml:"error,omitempty" raw:"column:error"`
}

// VersionsMarked возвращает версии через запятую, версия хранения отмечена *
func (i Item) VersionsMarked() string {
	if i.Versions == "" {
		return ""
	}
	parts := strings.Split(i.Versions, ",")
	for n, v := range parts {
		if v == i.Storage {
			parts[n] = v + "*"
		}
	}
	return strings.Join(parts, ",")
}

// defaultFormat таблица по умолчанию, пустые значения показываются прочерком
const defaultFormat = `table {{or .Name "-"}}\t{{or .Kind "-"}}\t{{or .Group "-"}}\t{{or .Plural "-"}}` +
	`\t{{or .Scope "-"}}\t{{or .VersionsMarked "-"}}\t{{.Status}}{{if .Error}}: {{.Error}}{{end}}`

// header подписи колонок для табличных форматов, ключи — имена полей и методов Item
var header = formatter.Header{
	"Name":           "NAME",
	"Kind":           "KIND",
	"Group":          "GROUP",
	"Plural":         "PLURAL",
	"Scope":          "SCOPE",
	"Versions":       "VERSIONS",
	"VersionsMarked": "VERSIONS",
	"Storage":        "STORAGE",
	"Status":         "STATUS",
}

// yamlFormat YAML с разделителем документов
// Встроенный формат yaml пакета formatter не разделяет элементы, и документы склеиваются
const yamlFormat = `---\n{{yaml .}}`

// format возвращает формат вывода по значению флага
func format(value string) formatter.Format {
	switch value {
	case "", formatter.TableFormatKey:
		return defaultFormat
	case formatter.YAMLFormatKey:
		return yamlFormat
	}
	return formatter.Format(value)
}

// printRows печатает список в заданном формате и возвращает число строк с ошибками
func printRows(w io.Writer, value string, rows []generate.Row) (int, error) {
	f, err := formatter.New(w)
	if err != nil {
		return 0, err
	}
	if err := f.SetFormat(format(value)); err != nil {
		return 0, fmt.Errorf("--%s: %w", formatFlag, err)
	}
	if err := f.SetHeader(header); err != nil {
		return 0, err
	}
	failed := 0
	for _, r := range rows {
		item := Item{
			Name:     r.Name,
			Kind:     r.Kind,
			Group:    r.Group,
			Plural:   r.Plural,
			Scope:    r.Scope,
			Versions: strings.Join(r.Versions, ","),
			Storage:  r.Storage,
			Status:   "ok",
			Error:    strings.ReplaceAll(r.Error, "\n", " "),
		}
		if item.Error != "" {
			item.Status = "error"
			failed++
		}
		if err := f.Write(item); err != nil {
			return failed, fmt.Errorf("--%s: %w", formatFlag, err)
		}
	}
	return failed, f.Flush()
}
