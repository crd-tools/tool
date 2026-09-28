package kubebuild

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-tools/pkg/crd"
	crdmarkers "sigs.k8s.io/controller-tools/pkg/crd/markers"
	"sigs.k8s.io/controller-tools/pkg/loader"
	"sigs.k8s.io/controller-tools/pkg/markers"

	"crd.tools/crd/registry"
	"crd.tools/gocmd"
)

// Generate строит схемы для типов и всех типов, на которые они ссылаются
// Ссылки в схемах переписываются в ключи реестра вида пакет.Тип
func Generate(ctx context.Context, root string, keys []string) (map[string]apiextv1.JSONSchemaProps, error) {
	byPkg := map[string][]string{}
	for _, key := range keys {
		pkg, name, err := registry.SplitKey(key)
		if err != nil {
			return nil, err
		}
		byPkg[pkg] = append(byPkg[pkg], name)
	}
	paths := make([]string, 0, len(byPkg))
	for path := range byPkg {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	roots, err := loader.LoadRootsWithConfig(&packages.Config{
		Context: ctx,
		Dir:     root,
		Env:     gocmd.Environ(),
	}, paths...)
	if err != nil {
		return nil, err
	}
	loaded := map[string]*loader.Package{}
	for _, pkg := range roots {
		loaded[loader.NonVendorPath(pkg.PkgPath)] = pkg
	}

	reg := &markers.Registry{}
	if err := crdmarkers.Register(reg); err != nil {
		return nil, err
	}
	parser := &crd.Parser{
		Collector: &markers.Collector{Registry: reg},
		Checker:   &loader.TypeChecker{},
	}
	crd.AddKnownTypes(parser)

	var idents []crd.TypeIdent
	for _, path := range paths {
		pkg, ok := loaded[path]
		if !ok {
			return nil, fmt.Errorf("package %s not found", path)
		}
		parser.NeedPackage(pkg)
		names := byPkg[path]
		sort.Strings(names)
		for _, name := range names {
			ident := crd.TypeIdent{Package: pkg, Name: name}
			parser.NeedSchemaFor(ident)
			idents = append(idents, ident)
		}
	}
	if err := collectErrors(roots); err != nil {
		return nil, err
	}
	return closure(parser, idents)
}

// closure обходит ссылки от корневых типов и переписывает их в ключи реестра
func closure(parser *crd.Parser, roots []crd.TypeIdent) (map[string]apiextv1.JSONSchemaProps, error) {
	idents := map[string]crd.TypeIdent{}
	for ident := range parser.Schemata {
		idents[identKey(ident)] = ident
	}

	out := map[string]apiextv1.JSONSchemaProps{}
	queue := append([]crd.TypeIdent(nil), roots...)
	for len(queue) > 0 {
		ident := queue[0]
		queue = queue[1:]
		key := identKey(ident)
		if _, done := out[key]; done {
			continue
		}
		raw, ok := parser.Schemata[ident]
		if !ok {
			return nil, fmt.Errorf("no schema for %s", key)
		}
		schema := *raw.DeepCopy()
		pkgPath := loader.NonVendorPath(ident.Package.PkgPath)

		var walkErr error
		registry.Walk(&schema, func(node *apiextv1.JSONSchemaProps) bool {
			if walkErr != nil || node.Ref == nil || *node.Ref == "" {
				return walkErr == nil
			}
			name, refPkg, err := crd.RefParts(*node.Ref)
			if err != nil {
				walkErr = fmt.Errorf("%s: %w", key, err)
				return false
			}
			if refPkg == "" {
				refPkg = pkgPath
			}
			refKey := registry.Key(loader.NonVendorPath(refPkg), name)
			target, ok := idents[refKey]
			if !ok {
				walkErr = fmt.Errorf("%s: missing schema for referenced type %s", key, refKey)
				return false
			}
			node.Ref = &refKey
			queue = append(queue, target)
			return true
		})
		if walkErr != nil {
			return nil, walkErr
		}
		out[key] = schema
	}
	return out, nil
}

func identKey(ident crd.TypeIdent) string {
	return registry.Key(loader.NonVendorPath(ident.Package.PkgPath), ident.Name)
}

// collectErrors собирает ошибки загруженных пакетов, как это делает controller-gen
// Ошибки проверки типов пропускаются: controller-tools проверяет типы частично
func collectErrors(roots []*loader.Package) error {
	seen := map[*loader.Package]bool{}
	var msgs []string
	var visit func(pkg *loader.Package)
	visit = func(pkg *loader.Package) {
		if seen[pkg] {
			return
		}
		seen[pkg] = true
		for _, e := range pkg.Errors {
			if e.Kind == packages.TypeError {
				continue
			}
			msgs = append(msgs, e.Error())
		}
		for _, imp := range pkg.Imports() {
			visit(imp)
		}
	}
	for _, pkg := range roots {
		visit(pkg)
	}
	if len(msgs) == 0 {
		return nil
	}
	sort.Strings(msgs)
	return errors.New("controller-tools:\n  " + strings.Join(uniqStrings(msgs), "\n  "))
}

func uniqStrings(items []string) []string {
	out := items[:0]
	for i, item := range items {
		if i > 0 && item == items[i-1] {
			continue
		}
		out = append(out, item)
	}
	return out
}
