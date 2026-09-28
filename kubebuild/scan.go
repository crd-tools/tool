package kubebuild

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"crd.tools/crd/registry"
	"crd.tools/gocmd"
)

// Tag это метка в теге поля, которая включает сбор схемы
type Tag struct {
	// Key ключ тега, например crd
	Key string

	// Value значение в списке через запятую, например kubebuild
	Value string
}

// String возвращает метку в виде тега поля
func (t Tag) String() string {
	return fmt.Sprintf("%s:%q", t.Key, t.Value)
}

// Check проверяет метку
func (t Tag) Check() error {
	if t.Key == "" || t.Value == "" {
		return errors.New("empty tag key or value")
	}
	if strings.ContainsAny(t.Key, " :\"`") || strings.ContainsAny(t.Value, " ,\"`") {
		return fmt.Errorf("invalid tag %s", t)
	}
	return nil
}

// Mark это найденная в коде метка
type Mark struct {
	// Type ключ типа поля, для которого нужна схема
	Type string

	// Usage место метки в виде пакет.Структура.Поле
	Usage string
}

// pending это метка, тип которой ещё не сопоставлен с путём импорта
type pending struct {
	pkg       string
	qualifier string
	name      string
	usage     string
	pos       string
	imports   []importSpec
}

type importSpec struct {
	alias string
	path  string
}

// ScanErrors это ошибки разбора меток с позициями в коде
type ScanErrors []string

func (e ScanErrors) Error() string {
	return "invalid marks:\n  " + strings.Join(e, "\n  ")
}

// Scan находит метки в файлах пакетов и возвращает типы, для которых нужны схемы
func Scan(ctx context.Context, dir string, pkgs []gocmd.Package, tag Tag) ([]Mark, error) {
	s := &scanner{tag: tag, fset: token.NewFileSet(), root: dir}
	for _, pkg := range pkgs {
		if err := s.scanPackage(pkg); err != nil {
			return nil, err
		}
	}
	marks, err := s.resolve(ctx)
	if err != nil {
		return nil, err
	}
	if len(s.errs) > 0 {
		sort.Strings(s.errs)
		return nil, s.errs
	}
	sort.Slice(marks, func(i, j int) bool {
		if marks[i].Type != marks[j].Type {
			return marks[i].Type < marks[j].Type
		}
		return marks[i].Usage < marks[j].Usage
	})
	return marks, nil
}

type scanner struct {
	tag     Tag
	fset    *token.FileSet
	root    string
	marks   []Mark
	pending []pending
	errs    ScanErrors
}

func (s *scanner) scanPackage(pkg gocmd.Package) error {
	quick := []byte(s.tag.Key + `:"`)
	for _, name := range pkg.GoFiles {
		file := filepath.Join(pkg.Dir, name)
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if !bytes.Contains(data, quick) {
			continue
		}
		f, err := parser.ParseFile(s.fset, file, data, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		s.scanFile(pkg.ImportPath, f)
	}
	return nil
}

func (s *scanner) scanFile(pkgPath string, f *ast.File) {
	var imports []importSpec
	for _, spec := range f.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		imp := importSpec{path: path}
		if spec.Name != nil {
			imp.alias = spec.Name.Name
		}
		imports = append(imports, imp)
	}
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		if st, ok := ts.Type.(*ast.StructType); ok {
			s.scanStruct(pkgPath, pkgPath+"."+ts.Name.Name, st, imports)
		}
		return true
	})
}

func (s *scanner) scanStruct(pkgPath, owner string, st *ast.StructType, imports []importSpec) {
	for _, field := range st.Fields.List {
		names := fieldNames(field)
		if nested, ok := field.Type.(*ast.StructType); ok {
			for _, name := range names {
				s.scanStruct(pkgPath, owner+"."+name, nested, imports)
			}
		}
		if !s.marked(field) {
			continue
		}
		for _, name := range names {
			s.addField(pkgPath, owner+"."+name, field, imports)
		}
	}
}

func fieldNames(field *ast.Field) []string {
	if len(field.Names) > 0 {
		out := make([]string, 0, len(field.Names))
		for _, n := range field.Names {
			out = append(out, n.Name)
		}
		return out
	}
	// встроенное поле называется по имени типа
	switch t := unwrap(field.Type).(type) {
	case *ast.Ident:
		return []string{t.Name}
	case *ast.SelectorExpr:
		return []string{t.Sel.Name}
	}
	return []string{"<embedded>"}
}

func (s *scanner) marked(field *ast.Field) bool {
	if field.Tag == nil {
		return false
	}
	raw, err := strconv.Unquote(field.Tag.Value)
	if err != nil {
		return false
	}
	value, ok := reflect.StructTag(raw).Lookup(s.tag.Key)
	if !ok {
		return false
	}
	for _, item := range strings.Split(value, ",") {
		if strings.TrimSpace(item) == s.tag.Value {
			return true
		}
	}
	return false
}

func (s *scanner) addField(pkgPath, usage string, field *ast.Field, imports []importSpec) {
	pos := s.position(field.Pos())
	switch t := unwrap(field.Type).(type) {
	case *ast.Ident:
		if types.Universe.Lookup(t.Name) != nil {
			s.errorf(pos, "%s: %s is a builtin type, %s requires a named struct type", usage, t.Name, s.tag)
			return
		}
		s.marks = append(s.marks, Mark{Type: registry.Key(pkgPath, t.Name), Usage: usage})
	case *ast.SelectorExpr:
		x, ok := t.X.(*ast.Ident)
		if !ok {
			s.errorf(pos, "%s: unsupported type expression", usage)
			return
		}
		s.pending = append(s.pending, pending{
			pkg:       pkgPath,
			qualifier: x.Name,
			name:      t.Sel.Name,
			usage:     usage,
			pos:       pos,
			imports:   imports,
		})
	case *ast.IndexExpr, *ast.IndexListExpr:
		s.errorf(pos, "%s: generic types are not supported", usage)
	case *ast.StructType:
		s.errorf(pos, "%s: anonymous struct types are not supported, use a named type", usage)
	case *ast.InterfaceType:
		s.errorf(pos, "%s: interface types have no schema", usage)
	default:
		s.errorf(pos, "%s: unsupported type expression %T", usage, t)
	}
}

// resolve сопоставляет квалификаторы типов с путями импорта
func (s *scanner) resolve(ctx context.Context) ([]Mark, error) {
	unnamed := map[string]struct{}{}
	for _, p := range s.pending {
		for _, imp := range p.imports {
			if imp.alias == "" {
				unnamed[imp.path] = struct{}{}
			}
		}
	}
	names := map[string]string{}
	if len(unnamed) > 0 {
		paths := make([]string, 0, len(unnamed))
		for path := range unnamed {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		pkgs, err := gocmd.List(ctx, s.root, paths...)
		if err != nil {
			return nil, err
		}
		for _, pkg := range pkgs {
			if pkg.Name != "" {
				names[pkg.ImportPath] = pkg.Name
			}
		}
	}

	marks := s.marks
	for _, p := range s.pending {
		path, ok := p.lookup(names)
		if !ok {
			s.errorf(p.pos, "%s: cannot resolve package %s", p.usage, p.qualifier)
			continue
		}
		marks = append(marks, Mark{Type: registry.Key(path, p.name), Usage: p.usage})
	}
	return marks, nil
}

func (p pending) lookup(names map[string]string) (string, bool) {
	for _, imp := range p.imports {
		name := imp.alias
		if name == "" {
			name = names[imp.path]
		}
		if name == p.qualifier {
			return imp.path, true
		}
	}
	return "", false
}

func (s *scanner) position(pos token.Pos) string {
	p := s.fset.Position(pos)
	file := p.Filename
	if rel, err := filepath.Rel(s.root, file); err == nil && !strings.HasPrefix(rel, "..") {
		file = rel
	}
	return fmt.Sprintf("%s:%d", filepath.ToSlash(file), p.Line)
}

func (s *scanner) errorf(pos, format string, args ...any) {
	s.errs = append(s.errs, pos+": "+fmt.Sprintf(format, args...))
}

// unwrap снимает указатели, срезы, массивы и значения мап до типа элемента
func unwrap(e ast.Expr) ast.Expr {
	for {
		switch t := e.(type) {
		case *ast.StarExpr:
			e = t.X
		case *ast.ParenExpr:
			e = t.X
		case *ast.ArrayType:
			e = t.Elt
		case *ast.MapType:
			e = t.Value
		default:
			return e
		}
	}
}
