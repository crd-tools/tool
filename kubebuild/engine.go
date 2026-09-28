package kubebuild

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"crd.tools/crd/registry"
	"crd.tools/gocmd"
)

// Settings это настройки движка
type Settings struct {
	// Tool имя инструмента для сообщений и заголовков файлов
	Tool string

	// Root корень модуля проекта
	Root string

	// ModulePath путь модуля проекта
	ModulePath string

	// Output каталог пакета со схемами относительно корня модуля
	Output string

	// Tag метка в тегах полей
	Tag Tag

	// Generator версия генератора, записывается в файлы
	Generator string
}

// Check проверяет настройки
func (s Settings) Check() error {
	switch {
	case s.Tool == "":
		return errors.New("empty tool name")
	case s.Root == "":
		return errors.New("empty module root")
	case s.ModulePath == "":
		return errors.New("empty module path")
	case s.Output == "":
		return errors.New("empty output")
	case s.Generator == "":
		return errors.New("empty generator version")
	}
	return s.Tag.Check()
}

// Report это итог работы get или prune
type Report struct {
	// Roots число типов, запрошенных метками
	Roots int

	// Types число типов в реестре вместе с зависимостями
	Types int

	// Packages число пакетов, из которых взяты типы
	Packages int

	// Generated запускался ли controller-tools
	Generated bool

	// Written записано файлов данных
	Written int

	// Removed удалено файлов данных
	Removed int

	// GoFile создан или обновлён Go-файл пакета
	GoFile bool

	// External шаблоны внешних пакетов из аргументов get
	External []string
}

// Print печатает отчёт
func (r Report) Print(w io.Writer, command string) {
	fmt.Fprintf(w, "%s: %d roots, %d types in %d packages\n", command, r.Roots, r.Types, r.Packages)
	if r.Generated {
		fmt.Fprintln(w, "  schemas regenerated with controller-tools")
	}
	fmt.Fprintf(w, "  files written: %d, removed: %d\n", r.Written, r.Removed)
	if r.GoFile {
		fmt.Fprintf(w, "  go file written: %s\n", GoFile)
	}
}

// Engine собирает схемы kubebuilder-типов в пакет проекта
type Engine struct {
	s Settings
}

// New возвращает движок
func New(s Settings) (*Engine, error) {
	if err := s.Check(); err != nil {
		return nil, err
	}
	return &Engine{s: s}, nil
}

// Get добавляет схемы для меток из пакетов по шаблонам, ничего не удаляя
// dir каталог, относительно которого разрешаются шаблоны
func (e *Engine) Get(ctx context.Context, dir string, patterns []string) (Report, error) {
	if len(patterns) == 0 {
		patterns = []string{"."}
	}
	marks, err := e.scan(ctx, dir, patterns)
	if err != nil {
		return Report{}, err
	}
	var external []string
	for _, p := range patterns {
		if isExternalPattern(p, e.s.ModulePath) {
			external = append(external, p)
		}
	}
	report, err := e.sync(ctx, marks, true)
	report.External = external
	return report, err
}

// Prune пересобирает реестр по всем шаблонам и удаляет неиспользуемые схемы
func (e *Engine) Prune(ctx context.Context, scan, external []string) (Report, error) {
	patterns := append(append([]string(nil), scan...), external...)
	marks, err := e.scan(ctx, e.s.Root, patterns)
	if err != nil {
		return Report{}, err
	}
	return e.sync(ctx, marks, false)
}

func (e *Engine) scan(ctx context.Context, dir string, patterns []string) ([]Mark, error) {
	pkgs, err := gocmd.List(ctx, dir, patterns...)
	if err != nil {
		return nil, err
	}
	output := filepath.Join(e.s.Root, filepath.FromSlash(e.s.Output))
	var errs []string
	scan := pkgs[:0]
	for _, pkg := range pkgs {
		if pkg.Error != nil {
			errs = append(errs, pkg.Error.Err)
			continue
		}
		if pkg.Dir == output {
			continue
		}
		scan = append(scan, pkg)
	}
	if len(errs) > 0 {
		return nil, errors.New("go list:\n  " + strings.Join(errs, "\n  "))
	}
	return Scan(ctx, e.s.Root, scan, e.s.Tag)
}

// sync приводит каталог данных к набору корневых типов
// keep добавляет к меткам корневые типы, уже записанные в реестр
func (e *Engine) sync(ctx context.Context, marks []Mark, keep bool) (Report, error) {
	dataDir := filepath.Join(e.s.Root, filepath.FromSlash(e.s.Output), DataDir)
	st, err := readStore(dataDir)
	if err != nil {
		return Report{}, err
	}

	roots := map[string][]string{}
	if keep {
		for _, r := range st.index.Roots {
			roots[r.Type] = append(roots[r.Type], r.Usages...)
		}
	}
	for _, m := range marks {
		roots[m.Type] = append(roots[m.Type], m.Usage)
	}
	keys := make([]string, 0, len(roots))
	for k := range roots {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pkgSet := map[string]struct{}{}
	for _, key := range keys {
		pkg, _, err := registry.SplitKey(key)
		if err != nil {
			return Report{}, err
		}
		pkgSet[pkg] = struct{}{}
	}
	for pkg := range st.files {
		pkgSet[pkg] = struct{}{}
	}
	vers, err := versions(ctx, e.s.Root, sortedKeys(pkgSet))
	if err != nil {
		return Report{}, err
	}

	report := Report{Roots: len(keys)}
	types, fresh := e.freshClosure(st, vers, keys)
	next := &store{files: map[string]*registry.File{}}
	if fresh {
		for key, t := range types {
			pkg, _, _ := registry.SplitKey(key)
			v := vers[pkg]
			e.file(next, pkg, v).Types = append(e.file(next, pkg, v).Types, t)
		}
	} else if len(keys) > 0 {
		schemas, err := Generate(ctx, e.s.Root, keys)
		if err != nil {
			return Report{}, err
		}
		report.Generated = true
		if err := e.fillGenerated(ctx, next, vers, schemas); err != nil {
			return Report{}, err
		}
	}

	next.index = registry.Index{Generator: e.s.Generator}
	for _, key := range keys {
		next.index.Roots = append(next.index.Roots, registry.Root{Type: key, Usages: uniq(roots[key])})
	}
	ch, err := next.write(dataDir)
	if err != nil {
		return Report{}, err
	}
	goFile, err := ensureGoFile(filepath.Dir(dataDir), e.s.Tool)
	if err != nil {
		return Report{}, err
	}

	report.Packages = len(next.files)
	for _, f := range next.files {
		report.Types += len(f.Types)
	}
	report.Written, report.Removed, report.GoFile = ch.written, ch.removed, goFile
	return report, nil
}

// freshClosure собирает типы от корней по существующим файлам
// Возвращает false, если какой-то тип отсутствует или его файл устарел
func (e *Engine) freshClosure(st *store, vers map[string]pkgVersion, roots []string) (map[string]registry.Type, bool) {
	out := map[string]registry.Type{}
	queue := append([]string(nil), roots...)
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		if _, done := out[key]; done {
			continue
		}
		f, t, ok := st.lookup(key)
		if !ok {
			return nil, false
		}
		v, known := vers[f.Package]
		if !known || v.module != f.Module || v.version != f.Version || f.Generator != e.s.Generator {
			return nil, false
		}
		out[key] = *t
		queue = append(queue, t.Refs...)
	}
	return out, true
}

// fillGenerated раскладывает сгенерированные схемы по файлам пакетов
func (e *Engine) fillGenerated(ctx context.Context, next *store, vers map[string]pkgVersion, schemas map[string]apiextv1.JSONSchemaProps) error {
	missing := map[string]struct{}{}
	for key := range schemas {
		pkg, _, err := registry.SplitKey(key)
		if err != nil {
			return err
		}
		if _, ok := vers[pkg]; !ok {
			missing[pkg] = struct{}{}
		}
	}
	if len(missing) > 0 {
		extra, err := versions(ctx, e.s.Root, sortedKeys(missing))
		if err != nil {
			return err
		}
		for pkg, v := range extra {
			vers[pkg] = v
		}
	}

	keys := make([]string, 0, len(schemas))
	for key := range schemas {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		pkg, name, _ := registry.SplitKey(key)
		v, ok := vers[pkg]
		if !ok {
			return fmt.Errorf("cannot determine version of package %s", pkg)
		}
		schema := schemas[key]
		data, err := schema.Marshal()
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		f := e.file(next, pkg, v)
		f.Types = append(f.Types, registry.Type{
			Name:   name,
			Refs:   registry.Refs(&schema),
			Schema: data,
		})
	}
	return nil
}

func (e *Engine) file(st *store, pkg string, v pkgVersion) *registry.File {
	f, ok := st.files[pkg]
	if !ok {
		f = &registry.File{
			Package:   pkg,
			Module:    v.module,
			Version:   v.version,
			Generator: e.s.Generator,
		}
		st.files[pkg] = f
	}
	return f
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func uniq(items []string) []string {
	sort.Strings(items)
	out := items[:0]
	for i, item := range items {
		if i > 0 && item == items[i-1] {
			continue
		}
		out = append(out, item)
	}
	return out
}
