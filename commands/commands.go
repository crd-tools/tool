package commands

import (
	"context"
	"os"
	"syscall"

	"github.com/mantyr/starter"
	"github.com/urfave/cli/v2"

	"crd.tools/components/kubebuild"
	"crd.tools/components/workspace"
	"crd.tools/logger"
)

// VerboseFlag включает подробный вывод инициализации компонентов
const VerboseFlag = "verbose"

// Settings это настройки инструмента, заданные в main.go
type Settings struct {
	// Name имя инструмента
	Name string

	// Env префикс переменных окружения
	Env string

	// Output каталог пакета со схемами по умолчанию
	Output string

	// OpenAPIFile имя файла настроек OpenAPI в корне модуля
	OpenAPIFile string

	// Workspace значения флагов workspace-компонента
	Workspace workspace.Defaults

	// Kubebuild значения флагов kubebuild-компонента
	Kubebuild kubebuild.Defaults
}

// Flags возвращает общие флаги всех команд
func Flags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    VerboseFlag,
			Aliases: []string{"v"},
			Usage:   "подробный вывод",
		},
	}
}

// Run инициализирует компоненты, выполняет fn и освобождает компоненты
// Ошибка уже напечатана логгером, поэтому наружу уходит только код выхода
func Run(ctx *cli.Context, fn starter.Func, components ...starter.Component) error {
	s, err := starter.New()
	if err != nil {
		return err
	}
	err = s.Logger(logger.New(ctx.App.ErrWriter, ctx.Bool(VerboseFlag))).
		Signals(os.Interrupt, syscall.SIGTERM).
		Init(ctx, components...).
		Run(ctx, fn).
		Stop().
		Wait(ctx).
		Error()
	if err != nil {
		return cli.Exit("", 1)
	}
	return nil
}

// Func это действие команды внутри Run
type Func = func(ctx *cli.Context, parent context.Context) error
