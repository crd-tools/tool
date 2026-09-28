package kubebuild

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"

	"crd.tools/gocmd"
)

// ToolModule путь модуля инструмента
const ToolModule = "crd.tools"

// controllerToolsModule путь модуля controller-tools
const controllerToolsModule = "sigs.k8s.io/controller-tools"

// GeneratorVersion возвращает версию генератора из сборки текущего бинарника
func GeneratorVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	versions := map[string]string{}
	add := func(m *debug.Module) {
		if m == nil {
			return
		}
		v := m.Version
		if m.Replace != nil {
			v = m.Replace.Version
			if v == "" {
				v = "(replaced)"
			}
		}
		versions[m.Path] = v
	}
	add(&info.Main)
	for _, dep := range info.Deps {
		add(dep)
	}
	return fmt.Sprintf("%s@%s %s@%s",
		ToolModule, orDevel(versions[ToolModule]),
		controllerToolsModule, orDevel(versions[controllerToolsModule]),
	)
}

func orDevel(v string) string {
	if v == "" {
		return "(devel)"
	}
	return v
}

// pkgVersion это версия пакета, по которой проверяется свежесть схем
type pkgVersion struct {
	module  string
	version string
}

// versions определяет версии пакетов по go.mod проекта
// Пакеты, которые не удалось найти, в результат не попадают
func versions(ctx context.Context, root string, paths []string) (map[string]pkgVersion, error) {
	out := map[string]pkgVersion{}
	if len(paths) == 0 {
		return out, nil
	}
	pkgs, err := gocmd.List(ctx, root, paths...)
	if err != nil {
		return nil, err
	}
	var goVersion string
	for _, pkg := range pkgs {
		if pkg.Error != nil || pkg.Dir == "" {
			continue
		}
		v, err := packageVersion(ctx, root, pkg, &goVersion)
		if err != nil {
			return nil, err
		}
		out[pkg.ImportPath] = v
	}
	return out, nil
}

func packageVersion(ctx context.Context, root string, pkg gocmd.Package, goVersion *string) (pkgVersion, error) {
	if pkg.Standard {
		if *goVersion == "" {
			v, err := gocmd.Env(ctx, root, "GOVERSION")
			if err != nil {
				return pkgVersion{}, err
			}
			*goVersion = v
		}
		return pkgVersion{module: "std", version: *goVersion}, nil
	}
	m := pkg.Module
	if m == nil {
		return sourceVersion("", pkg)
	}
	if m.Replace != nil {
		if m.Replace.Version == "" {
			return sourceVersion(m.Path, pkg)
		}
		return pkgVersion{module: m.Path, version: m.Replace.Path + "@" + m.Replace.Version}, nil
	}
	if m.Main || m.Version == "" {
		return sourceVersion(m.Path, pkg)
	}
	return pkgVersion{module: m.Path, version: m.Version}, nil
}

// sourceVersion считает версию локального пакета по содержимому исходников
func sourceVersion(module string, pkg gocmd.Package) (pkgVersion, error) {
	files := append([]string(nil), pkg.GoFiles...)
	sort.Strings(files)
	h := sha256.New()
	for _, name := range files {
		data, err := os.ReadFile(filepath.Join(pkg.Dir, name))
		if err != nil {
			return pkgVersion{}, err
		}
		fmt.Fprintf(h, "%s\x00%d\x00", name, len(data))
		h.Write(data)
	}
	sum := hex.EncodeToString(h.Sum(nil))
	return pkgVersion{module: module, version: "src:" + sum[:32]}, nil
}

// isExternalPattern сообщает, указывает ли шаблон на пакеты вне модуля проекта
func isExternalPattern(pattern, modulePath string) bool {
	if pattern == "" || strings.HasPrefix(pattern, ".") || filepath.IsAbs(pattern) {
		return false
	}
	return pattern != modulePath && !strings.HasPrefix(pattern, modulePath+"/")
}
