package gocmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Package это сведения о пакете из go list
type Package struct {
	ImportPath string
	Name       string
	Dir        string
	GoFiles    []string
	Standard   bool
	Module     *Module
	Error      *PackageError
}

// Module это сведения о модуле пакета
type Module struct {
	Path    string
	Version string
	Main    bool
	Dir     string
	Replace *Module
}

// PackageError ошибка загрузки пакета
type PackageError struct {
	Err string
}

// listFields поля, которые запрашиваются у go list
const listFields = "ImportPath,Name,Dir,GoFiles,Standard,Module,Error"

// Env возвращает значение переменной окружения Go
func Env(ctx context.Context, dir, key string) (string, error) {
	out, err := run(ctx, dir, "env", key)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// List возвращает пакеты по шаблонам, не разрешая их зависимости
// Ошибки отдельных пакетов возвращаются в поле Error, а не общей ошибкой
func List(ctx context.Context, dir string, patterns ...string) ([]Package, error) {
	if len(patterns) == 0 {
		return nil, nil
	}
	args := append([]string{"list", "-e", "-find", "-json=" + listFields, "--"}, patterns...)
	out, err := run(ctx, dir, args...)
	if err != nil {
		return nil, err
	}
	var pkgs []Package
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var p Package
		err := dec.Decode(&p)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("go list: decode: %w", err)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

func run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Env = Environ()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("go %s: %s", args[0], msg)
	}
	return out, nil
}

// Environ возвращает окружение для дочерних вызовов go
// Флаг -modfile убирается из GOFLAGS, чтобы пакеты проекта грузились по его go.mod
func Environ() []string {
	env := os.Environ()
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if value, ok := strings.CutPrefix(kv, "GOFLAGS="); ok {
			kv = "GOFLAGS=" + stripModfile(value)
		}
		out = append(out, kv)
	}
	return out
}

func stripModfile(flags string) string {
	fields := strings.Fields(flags)
	out := fields[:0]
	for _, f := range fields {
		if strings.HasPrefix(f, "-modfile=") || strings.HasPrefix(f, "--modfile=") {
			continue
		}
		out = append(out, f)
	}
	return strings.Join(out, " ")
}
