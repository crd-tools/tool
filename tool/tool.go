package tool

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	app "github.com/mantyr/app"
	log "github.com/sirupsen/logrus"

	"crd.tools/commands"
	generatecmd "crd.tools/commands/generate"
	"crd.tools/commands/initialize"
	kubebuildcmd "crd.tools/commands/kubebuild"
	listcmd "crd.tools/commands/list"
	openapicmd "crd.tools/commands/openapi"
	kubebuildcomponent "crd.tools/components/kubebuild"
	"crd.tools/components/workspace"
	"crd.tools/kubebuild"
)

// Config это настройки инструмента, задаются в main.go
type Config struct {
	// Name команда вызова для справки и сообщений, например go tool crd.tools
	Name string

	// Env префикс переменных окружения, например CRD_TOOLS
	Env string

	// Usage краткое описание для справки
	Usage string

	// ConfigFile имя файла настроек проекта в корне модуля
	ConfigFile string

	// Output каталог пакета со схемами по умолчанию для init
	Output string

	// OpenAPIFile имя файла настроек OpenAPI в корне модуля
	OpenAPIFile string

	// Kubebuild настройки сбора схем из kubebuilder-маркеров
	Kubebuild Kubebuild
}

// envPattern допустимый префикс переменных окружения
var envPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// Kubebuild это настройки сбора схем из kubebuilder-маркеров
type Kubebuild struct {
	// TagKey ключ тега поля, например crd
	TagKey string

	// TagValue значение метки в теге поля, например kubebuild
	TagValue string
}

// Check проверяет настройки
func (c Config) Check() error {
	switch {
	case c.Name == "":
		return errors.New("empty Name")
	case !envPattern.MatchString(c.Env):
		return fmt.Errorf("bad Env %q, want %s", c.Env, envPattern)
	case c.ConfigFile == "":
		return errors.New("empty ConfigFile")
	case c.Output == "":
		return errors.New("empty Output")
	case c.OpenAPIFile == "":
		return errors.New("empty OpenAPIFile")
	}
	return kubebuild.Tag{Key: c.Kubebuild.TagKey, Value: c.Kubebuild.TagValue}.Check()
}

// Tool это консольный инструмент
type Tool struct {
	config Config
}

// New возвращает инструмент
func New(c Config) *Tool {
	return &Tool{config: c}
}

// RunAndFatal запускает инструмент и завершает процесс при ошибке
func (t *Tool) RunAndFatal(args []string) {
	if err := t.config.Check(); err != nil {
		log.Fatal(fmt.Errorf("tool config: %w", err))
	}
	settings := t.settings()

	a := app.New()
	a.Name = t.config.Name
	a.Usage = t.config.Usage
	a.Version = version()
	a.Register(
		initialize.New(settings),
		kubebuildcmd.New(settings),
		generatecmd.New(settings),
		listcmd.New(settings),
		openapicmd.New(settings),
	)
	a.RunAndFatal(args)
}

func (t *Tool) settings() commands.Settings {
	env := t.config.Env
	return commands.Settings{
		Name:        t.config.Name,
		Env:         env,
		Output:      t.config.Output,
		OpenAPIFile: t.config.OpenAPIFile,
		Workspace: workspace.Defaults{
			Env:        env,
			ConfigFile: t.config.ConfigFile,
		},
		Kubebuild: kubebuildcomponent.Defaults{
			Env: env,
			Tag: kubebuild.Tag{
				Key:   t.config.Kubebuild.TagKey,
				Value: t.config.Kubebuild.TagValue,
			},
		},
	}
}

// version возвращает версию модуля инструмента из сборки
func version() string {
	v := kubebuild.GeneratorVersion()
	if tool, _, ok := strings.Cut(v, " "); ok {
		return strings.TrimPrefix(tool, kubebuild.ToolModule+"@")
	}
	return v
}
