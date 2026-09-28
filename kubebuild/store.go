package kubebuild

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"crd.tools/crd/registry"
)

// DataDir имя каталога с данными внутри пакета со схемами
const DataDir = "data"

// store это содержимое каталога данных в памяти
type store struct {
	index registry.Index
	files map[string]*registry.File
}

// changes это итог записи каталога данных
type changes struct {
	written   int
	removed   int
	unchanged int
}

// readStore читает каталог данных, отсутствующий каталог даёт пустое хранилище
func readStore(dir string) (*store, error) {
	s := &store{files: map[string]*registry.File{}}
	data, err := os.ReadFile(filepath.Join(dir, registry.IndexFile))
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	idx, err := registry.UnmarshalIndex(data)
	if errors.Is(err, registry.ErrFormat) {
		// снимки другой версии формата перегенерируются целиком
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", registry.IndexFile, err)
	}
	s.index = idx
	for _, name := range idx.Files {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		f, err := registry.UnmarshalFile(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		s.files[f.Package] = &f
	}
	return s, nil
}

// lookup возвращает схему типа по ключу
func (s *store) lookup(key string) (*registry.File, *registry.Type, bool) {
	pkg, name, err := registry.SplitKey(key)
	if err != nil {
		return nil, nil, false
	}
	f, ok := s.files[pkg]
	if !ok {
		return nil, nil, false
	}
	i := sort.Search(len(f.Types), func(i int) bool { return f.Types[i].Name >= name })
	if i == len(f.Types) || f.Types[i].Name != name {
		return f, nil, false
	}
	return f, &f.Types[i], true
}

// write записывает каталог данных, не трогая файлы с неизменным содержимым
func (s *store) write(dir string) (changes, error) {
	var ch changes
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ch, err
	}

	pkgs := make([]string, 0, len(s.files))
	for pkg := range s.files {
		pkgs = append(pkgs, pkg)
	}
	sort.Strings(pkgs)

	keep := map[string]bool{registry.IndexFile: true}
	s.index.Files = s.index.Files[:0]
	s.index.Types = s.index.Types[:0]
	for _, pkg := range pkgs {
		f := s.files[pkg]
		sort.Slice(f.Types, func(i, j int) bool { return f.Types[i].Name < f.Types[j].Name })
		data, err := registry.MarshalFile(*f)
		if err != nil {
			return ch, fmt.Errorf("%s: %w", pkg, err)
		}
		name := registry.FileName(pkg)
		keep[name] = true
		s.index.Files = append(s.index.Files, name)
		for _, t := range f.Types {
			s.index.Types = append(s.index.Types, registry.Entry{
				Type: registry.Key(pkg, t.Name),
				File: name,
				Hash: registry.Hash(t.Schema),
			})
		}
		if err := writeIfChanged(filepath.Join(dir, name), data, &ch); err != nil {
			return ch, err
		}
	}

	sort.Slice(s.index.Types, func(i, j int) bool { return s.index.Types[i].Type < s.index.Types[j].Type })
	data, err := registry.MarshalIndex(s.index)
	if err != nil {
		return ch, err
	}
	if err := writeIfChanged(filepath.Join(dir, registry.IndexFile), data, &ch); err != nil {
		return ch, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return ch, err
	}
	for _, e := range entries {
		if e.IsDir() || keep[e.Name()] || !strings.HasSuffix(e.Name(), registry.FileExt) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			return ch, err
		}
		ch.removed++
	}
	return ch, nil
}

// writeIfChanged пишет файл атомарно и только если содержимое отличается
func writeIfChanged(file string, data []byte, ch *changes) error {
	cur, err := os.ReadFile(file)
	if err == nil && bytes.Equal(cur, data) {
		ch.unchanged++
		return nil
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	tmp := file + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, file); err != nil {
		os.Remove(tmp)
		return err
	}
	ch.written++
	return nil
}
