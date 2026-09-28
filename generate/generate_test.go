package generate

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func writeFile(t *testing.T, file, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscover(t *testing.T) {
	Convey("Discover finds packages with crd.Add in init", t, func() {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "go.mod"), "module example.com/m\n\ngo 1.24\n")
		// обычный импорт
		writeFile(t, filepath.Join(root, "a/a.go"), `package a
import "crd.tools/crd"
func init() { crd.Add(nil) }
`)
		// алиас импорта
		writeFile(t, filepath.Join(root, "b/b.go"), `package b
import c "crd.tools/crd"
func init() { if true { c.Add(nil) } }
`)
		// вызов вне init не считается
		writeFile(t, filepath.Join(root, "c/c.go"), `package c
import "crd.tools/crd"
func Register() { crd.Add(nil) }
`)
		// пакет main нельзя импортировать
		writeFile(t, filepath.Join(root, "cmd/x/main.go"), `package main
import "crd.tools/crd"
func init() { crd.Add(nil) }
func main() {}
`)
		// Add из другого пакета
		writeFile(t, filepath.Join(root, "d/d.go"), `package d
import crd "example.com/other"
func init() { crd.Add(nil) }
`)
		pkgs, err := Discover(context.Background(), root, []string{"./..."})
		So(err, ShouldBeNil)
		So(pkgs, ShouldResemble, []string{"example.com/m/a", "example.com/m/b"})
	})
}

func TestRenderMain(t *testing.T) {
	Convey("renderMain", t, func() {
		Convey("imports packages for their init and the snapshots package", func() {
			src := string(renderMain("example.com/m/schemas", []string{"example.com/m/a"}, Filter{Kind: "App", Group: "example.com"}, Overrides{}))
			So(src, ShouldContainSubstring, `_ "example.com/m/a"`)
			So(src, ShouldContainSubstring, `crd.SetDERSource(schemas.FS())`)
			So(src, ShouldContainSubstring, `c.GroupVersionKind("").Kind != "App"`)
			So(src, ShouldContainSubstring, `c.GroupVersionKind("").Group != "example.com"`)
			So(src, ShouldNotContainSubstring, `Resource != `)
			So(src, ShouldNotContainSubstring, "strconv")
		})

		Convey("applies overrides as Go literals", func() {
			src := string(renderMain("", []string{"example.com/m/a"}, Filter{}, Overrides{
				Group:  `ex"ample.com`,
				Labels: map[string]string{"b": "2", "a": "1"},
			}))
			So(src, ShouldContainSubstring, `c.Group("ex\"ample.com")`)
			So(strings.Index(src, `c.Label("a", "1")`), ShouldBeLessThan, strings.Index(src, `c.Label("b", "2")`))
			So(src, ShouldContainSubstring, "len(selected) != 1")
		})
	})
}

func TestWriteFiles(t *testing.T) {
	Convey("writeFiles numbers name collisions and warns", t, func() {
		obj := func(kind string) *apiextv1.CustomResourceDefinition {
			return &apiextv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{Name: "apps.example.com"},
				Spec: apiextv1.CustomResourceDefinitionSpec{
					Group: "example.com",
					Names: apiextv1.CustomResourceDefinitionNames{Kind: kind, Plural: "apps"},
				},
			}
		}
		dir := t.TempDir()
		var stderr bytes.Buffer
		files, err := writeFiles(dir, &stderr, []*apiextv1.CustomResourceDefinition{obj("A"), obj("B"), obj("C")})
		So(err, ShouldBeNil)
		So(files, ShouldResemble, []string{
			filepath.Join(dir, "example.com_apps.yaml"),
			filepath.Join(dir, "example.com_apps_1.yaml"),
			filepath.Join(dir, "example.com_apps_2.yaml"),
		})
		So(strings.Count(stderr.String(), "warning:"), ShouldEqual, 2)

		data, err := os.ReadFile(files[0])
		So(err, ShouldBeNil)
		So(string(data), ShouldStartWith, fileHeader+"---\n")
	})
}

func TestFilter(t *testing.T) {
	Convey("Filter", t, func() {
		So(Filter{}.Empty(), ShouldBeTrue)
		So(Filter{Plural: "apps"}.Empty(), ShouldBeFalse)
		So(Filter{Kind: "App", Group: "example.com", Plural: "apps"}.String(), ShouldEqual, "kind=App, group=example.com, plural=apps")

		src := string(renderMain("", []string{"example.com/m/a"}, Filter{Plural: "apps"}, Overrides{}))
		So(src, ShouldContainSubstring, `c.GroupVersionResource("").Resource != "apps"`)
	})
}
