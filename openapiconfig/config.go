package openapiconfig

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"

	"gopkg.in/yaml.v3"

	"crd.tools/crd"
)

// Version версия формата файла
const Version = 1

// Config это настройки генерации OpenAPI
type Config struct {
	// Version версия формата файла
	Version int `yaml:"version"`

	// Info заголовок и версия документа
	Info Info `yaml:"info,omitempty"`

	// Prefixes замены начала имён компонентов, отсортированы по From
	Prefixes []crd.Prefix `yaml:"prefixes,omitempty"`
}

// Info это раздел info документа
type Info struct {
	Title   string `yaml:"title,omitempty"`
	Version string `yaml:"version,omitempty"`
}

// Load читает файл настроек, отсутствующий файл даёт пустые настройки
func Load(file string) (*Config, error) {
	data, err := os.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return &Config{Version: Version}, nil
	}
	if err != nil {
		return nil, err
	}
	c := &Config{}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(c); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	if c.Version != Version {
		return nil, fmt.Errorf("%s: unsupported version %d", file, c.Version)
	}
	return c, nil
}

// Save записывает файл настроек
func (c *Config) Save(file string) error {
	c.Version = Version
	c.sort()
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

// Add добавляет замену префикса или меняет существующую с тем же From
func (c *Config) Add(from, to string) error {
	if from == "" {
		return errors.New("empty prefix")
	}
	for i, p := range c.Prefixes {
		if p.From == from {
			c.Prefixes[i].To = to
			return nil
		}
	}
	c.Prefixes = append(c.Prefixes, crd.Prefix{From: from, To: to})
	c.sort()
	return nil
}

// Remove удаляет замену префикса и сообщает, была ли она
func (c *Config) Remove(from string) bool {
	for i, p := range c.Prefixes {
		if p.From == from {
			c.Prefixes = append(c.Prefixes[:i], c.Prefixes[i+1:]...)
			return true
		}
	}
	return false
}

func (c *Config) sort() {
	sort.Slice(c.Prefixes, func(i, j int) bool { return c.Prefixes[i].From < c.Prefixes[j].From })
}
