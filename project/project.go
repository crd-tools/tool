package project

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Version версия формата файла настроек
const Version = 1

// Config это настройки инструмента в проекте
type Config struct {
	// Version версия формата файла
	Version int `yaml:"version"`

	// Output каталог пакета со схемами относительно корня модуля
	Output string `yaml:"output"`

	// Kubebuild настройки сбора схем из kubebuilder-маркеров
	Kubebuild Kubebuild `yaml:"kubebuild"`
}

// Kubebuild это настройки сбора схем из kubebuilder-маркеров
type Kubebuild struct {
	// Scan шаблоны пакетов модуля, которые обходит prune
	Scan []string `yaml:"scan"`

	// External внешние пакеты, добавленные через get, которые тоже обходит prune
	External []string `yaml:"external,omitempty"`
}

// New возвращает настройки по умолчанию с указанным каталогом вывода
func New(output string) *Config {
	return &Config{
		Version: Version,
		Output:  output,
		Kubebuild: Kubebuild{
			Scan: []string{"./..."},
		},
	}
}

// Load читает файл настроек
func Load(file string) (*Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	c := &Config{}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(c); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	if err := c.Check(); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return c, nil
}

// Save записывает файл настроек
func (c *Config) Save(file string) error {
	if err := c.Check(); err != nil {
		return err
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(c); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return os.WriteFile(file, buf.Bytes(), 0o644)
}

// Check проверяет настройки
func (c *Config) Check() error {
	if c.Version != Version {
		return fmt.Errorf("unsupported config version %d", c.Version)
	}
	if c.Output == "" {
		return errors.New("empty output")
	}
	if path.IsAbs(c.Output) || c.Output == ".." || strings.HasPrefix(c.Output, "../") {
		return fmt.Errorf("output must be inside the module: %s", c.Output)
	}
	if len(c.Kubebuild.Scan) == 0 {
		return errors.New("empty kubebuild.scan")
	}
	return nil
}

// AddExternal добавляет внешние шаблоны пакетов и сообщает, изменился ли список
func (c *Config) AddExternal(patterns ...string) bool {
	set := map[string]struct{}{}
	for _, p := range c.Kubebuild.External {
		set[p] = struct{}{}
	}
	changed := false
	for _, p := range patterns {
		if _, ok := set[p]; ok {
			continue
		}
		set[p] = struct{}{}
		c.Kubebuild.External = append(c.Kubebuild.External, p)
		changed = true
	}
	sort.Strings(c.Kubebuild.External)
	return changed
}
