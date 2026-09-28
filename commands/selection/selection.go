package selection

import (
	"os"
	"path"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"crd.tools/components/workspace"
	"crd.tools/generate"
	"crd.tools/kubebuild"
)

// Флаги отбора определений, общие для generate и list
const (
	KindFlag     = "kind"
	GroupFlag    = "group"
	PluralFlag   = "plural"
	ValidateFlag = "validate"
)

// Flags возвращает флаги отбора и проверки
func Flags() []cli.Flag {
	return append(FilterFlags(),
		&cli.BoolFlag{Name: ValidateFlag, Usage: "проверить каждый CRD кодом API-сервера"},
	)
}

// FilterFlags возвращает флаги отбора
func FilterFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: KindFlag, Usage: "оставить только CRD с этим kind"},
		&cli.StringFlag{Name: GroupFlag, Usage: "оставить только CRD с этой group"},
		&cli.StringFlag{Name: PluralFlag, Usage: "оставить только CRD с этим plural"},
	}
}

// Filter читает фильтр из флагов
func Filter(ctx *cli.Context) generate.Filter {
	return generate.Filter{
		Kind:   ctx.String(KindFlag),
		Group:  ctx.String(GroupFlag),
		Plural: ctx.String(PluralFlag),
	}
}

// SchemasPackage возвращает путь импорта пакета со снимками, если он уже сгенерирован
func SchemasPackage(ws workspace.Workspace) string {
	p, err := ws.Project()
	if err != nil {
		return ""
	}
	if _, err := os.Stat(filepath.Join(ws.Root(), filepath.FromSlash(p.Output), kubebuild.GoFile)); err != nil {
		return ""
	}
	return path.Join(ws.ModulePath(), p.Output)
}

// Patterns возвращает шаблоны пакетов и проверяет, что среди них нет флагов
// urfave/cli v2 разбирает флаги только до первого позиционного аргумента
func Patterns(ctx *cli.Context, command string) ([]string, error) {
	patterns := ctx.Args().Slice()
	for _, p := range patterns {
		if len(p) > 0 && p[0] == '-' {
			return nil, cli.Exit("flag "+p+" after package patterns: flags must come first: "+command+" [flags] [packages]", 2)
		}
	}
	return patterns, nil
}
