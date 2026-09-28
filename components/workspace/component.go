package workspace

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
	"golang.org/x/mod/modfile"

	"crd.tools/gocmd"
	"crd.tools/project"
)

// ErrNotInitialized в модуле нет файла настроек инструмента
var ErrNotInitialized = errors.New("project is not initialized")

// Workspace это модуль проекта, в котором запущен инструмент
type Workspace interface {
	// Root возвращает корень модуля
	Root() string

	// ModulePath возвращает путь модуля
	ModulePath() string

	// ConfigFile возвращает абсолютный путь к файлу настроек
	ConfigFile() string

	// Exists сообщает, есть ли файл настроек
	Exists() bool

	// Project возвращает настройки проекта
	//   ErrNotInitialized
	Project() (*project.Config, error)

	// Save записывает настройки проекта
	Save(c *project.Config) error
}

// Component это компонент рабочего модуля
type Component struct {
	dep        Dep
	config     Config
	root       string
	modulePath string
	configFile string
	project    *project.Config
}

// New возвращает компонент
func New(dep Dep) *Component {
	return &Component{dep: dep}
}

// Name возвращает имя компонента
func (c *Component) Name() string {
	return "Workspace"
}

// Init находит корень модуля и читает файл настроек, если он есть
func (c *Component) Init(ctx *cli.Context) error {
	if c.dep == nil {
		return errors.New("empty dep")
	}
	c.config = Parse(ctx)
	if c.config.ConfigFile == "" {
		return errors.New("empty " + ConfigFileFlag)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	gomod, err := gocmd.Env(ctx.Context, cwd, "GOMOD")
	if err != nil {
		return err
	}
	if gomod == "" || gomod == os.DevNull {
		return errors.New("not inside a Go module")
	}
	data, err := os.ReadFile(gomod)
	if err != nil {
		return err
	}
	c.root = filepath.Dir(gomod)
	c.modulePath = modfile.ModulePath(data)
	if c.modulePath == "" {
		return fmt.Errorf("%s: empty module path", gomod)
	}

	c.configFile = c.config.ConfigFile
	if !filepath.IsAbs(c.configFile) {
		c.configFile = filepath.Join(c.root, c.configFile)
	}
	c.project, err = project.Load(c.configFile)
	if errors.Is(err, fs.ErrNotExist) {
		c.project = nil
		return nil
	}
	return err
}

// Destroy ничего не освобождает
func (c *Component) Destroy(ctx *cli.Context) error {
	return nil
}

// Root возвращает корень модуля
func (c *Component) Root() string {
	return c.root
}

// ModulePath возвращает путь модуля
func (c *Component) ModulePath() string {
	return c.modulePath
}

// ConfigFile возвращает абсолютный путь к файлу настроек
func (c *Component) ConfigFile() string {
	return c.configFile
}

// Exists сообщает, есть ли файл настроек
func (c *Component) Exists() bool {
	return c.project != nil
}

// Project возвращает настройки проекта
//
//	ErrNotInitialized
func (c *Component) Project() (*project.Config, error) {
	if c.project == nil {
		return nil, fmt.Errorf("%w: %s not found", ErrNotInitialized, c.configFile)
	}
	return c.project, nil
}

// Save записывает настройки проекта
func (c *Component) Save(p *project.Config) error {
	if err := p.Save(c.configFile); err != nil {
		return err
	}
	c.project = p
	return nil
}
