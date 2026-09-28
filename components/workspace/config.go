package workspace

import (
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

// Флаги компонента
const (
	// ConfigFileFlag путь к файлу настроек проекта относительно корня модуля
	ConfigFileFlag = "tool.config"
)

// Defaults это значения флагов по умолчанию, задаются в main.go инструмента
type Defaults struct {
	// Env префикс переменных окружения
	Env string

	// ConfigFile имя файла настроек проекта
	ConfigFile string
}

// Config это настройки компонента
type Config struct {
	// ConfigFile путь к файлу настроек проекта
	ConfigFile string
}

// Flags возвращает флаги компонента
func Flags(d Defaults) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    ConfigFileFlag,
			Usage:   "файл настроек проекта относительно корня модуля",
			EnvVars: []string{d.Env + "_CONFIG"},
			Value:   d.ConfigFile,
		}),
	}
}

// Parse читает настройки компонента из флагов
func Parse(ctx *cli.Context) Config {
	return Config{
		ConfigFile: ctx.String(ConfigFileFlag),
	}
}
