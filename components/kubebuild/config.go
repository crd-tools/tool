package kubebuild

import (
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"

	"crd.tools/kubebuild"
)

// Флаги компонента
const (
	// TagKeyFlag ключ тега поля с меткой
	TagKeyFlag = "kubebuild.tag.key"

	// TagValueFlag значение метки в теге поля
	TagValueFlag = "kubebuild.tag.value"
)

// Defaults это значения флагов по умолчанию, задаются в main.go инструмента
type Defaults struct {
	// Env префикс переменных окружения
	Env string

	// Tag метка в тегах полей
	Tag kubebuild.Tag
}

// Config это настройки компонента
type Config struct {
	// Tag метка в тегах полей
	Tag kubebuild.Tag
}

// Flags возвращает флаги компонента
func Flags(d Defaults) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    TagKeyFlag,
			Usage:   "ключ тега поля, в котором ищется метка",
			EnvVars: []string{d.Env + "_KUBEBUILD_TAG_KEY"},
			Value:   d.Tag.Key,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    TagValueFlag,
			Usage:   "значение метки в теге поля",
			EnvVars: []string{d.Env + "_KUBEBUILD_TAG_VALUE"},
			Value:   d.Tag.Value,
		}),
	}
}

// Parse читает настройки компонента из флагов
func Parse(ctx *cli.Context) Config {
	return Config{
		Tag: kubebuild.Tag{
			Key:   ctx.String(TagKeyFlag),
			Value: ctx.String(TagValueFlag),
		},
	}
}
