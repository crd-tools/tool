package kubebuild

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"crd.tools/crd/registry"
)

const demoLegacy = `package v1

// Common это общие поля
type Common struct {
	// Owner владелец
	// +kubebuilder:validation:MinLength=1
	Owner string ` + "`json:\"owner\"`" + `
}

// Resources это ресурсы
// +kubebuilder:validation:XValidation:rule="self.limits.cpu >= self.requests.cpu",message="limits must be >= requests"
type Resources struct {
	Common ` + "`json:\",inline\"`" + `

	// Requests минимальные ресурсы
	Requests ResourceList ` + "`json:\"requests\"`" + `

	// Limits максимальные ресурсы
	// +optional
	Limits *ResourceList ` + "`json:\"limits,omitempty\"`" + `

	// Priority приоритет
	// +kubebuilder:validation:Enum=low;normal;high
	// +kubebuilder:default=normal
	Priority string ` + "`json:\"priority,omitempty\"`" + `
}

// ResourceList это набор ресурсов
type ResourceList struct {
	// +kubebuilder:validation:Minimum=0
	CPU int64 ` + "`json:\"cpu\"`" + `
}
`

const demoAPI = `package api

import (
	legacy "example.com/demo/legacy/v1"
)

// Spec это спецификация
type Spec struct {
	Name      string            ` + "`json:\"name\"`" + `
	Resources legacy.Resources  ` + "`json:\"resources\" crd:\"kubebuild\"`" + `
	Extra     []legacy.Resources ` + "`json:\"extra\" crd:\"other,kubebuild\"`" + `
}
`

func writeFile(t *testing.T, file, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func demoModule(t *testing.T) string {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/demo\n\ngo 1.24\n")
	writeFile(t, filepath.Join(root, "legacy/v1/types.go"), demoLegacy)
	writeFile(t, filepath.Join(root, "api/spec.go"), demoAPI)
	return root
}

func TestEngine(t *testing.T) {
	if testing.Short() {
		t.Skip("uses go toolchain")
	}
	ctx := context.Background()

	Convey("Engine on a demo module", t, func() {
		root := demoModule(t)
		e, err := New(Settings{
			Tool:       "go tool crd.tools",
			Root:       root,
			ModulePath: "example.com/demo",
			Output:     "internal/schemas",
			Tag:        Tag{Key: "crd", Value: "kubebuild"},
			Generator:  "test",
		})
		So(err, ShouldBeNil)
		out := filepath.Join(root, "internal/schemas")

		Convey("get collects schemas with their dependencies", func() {
			report, err := e.Get(ctx, root, []string{"./..."})
			So(err, ShouldBeNil)
			So(report.Roots, ShouldEqual, 1)
			So(report.Types, ShouldEqual, 3)
			So(report.Generated, ShouldBeTrue)
			So(report.GoFile, ShouldBeTrue)
			_, statErr := os.Stat(filepath.Join(out, GoFile))
			So(statErr, ShouldBeNil)

			reg := registry.New()
			So(reg.Load(os.DirFS(out), DataDir), ShouldBeNil)
			So(reg.Keys(), ShouldResemble, []string{
				"example.com/demo/legacy/v1.Common",
				"example.com/demo/legacy/v1.ResourceList",
				"example.com/demo/legacy/v1.Resources",
			})
			roots := reg.Roots()
			So(roots, ShouldHaveLength, 1)
			So(roots[0].Usages, ShouldResemble, []string{
				"example.com/demo/api.Spec.Extra",
				"example.com/demo/api.Spec.Resources",
			})

			raw, err := reg.Get("example.com/demo/legacy/v1.Resources")
			So(err, ShouldBeNil)
			So(*raw.Properties["requests"].Ref, ShouldEqual, "example.com/demo/legacy/v1.ResourceList")

			s, err := reg.Resolve("example.com/demo/legacy/v1.Resources")
			So(err, ShouldBeNil)
			So(s.AllOf, ShouldBeEmpty)
			So(s.Properties, ShouldContainKey, "owner")
			So(s.Required, ShouldResemble, []string{"owner", "requests"})
			So(s.Properties["requests"].Description, ShouldEqual, "Requests минимальные ресурсы")
			So(*s.Properties["requests"].Properties["cpu"].Minimum, ShouldEqual, 0)
			So(s.Properties["priority"].Enum, ShouldHaveLength, 3)
			So(s.XValidations, ShouldHaveLength, 1)

			Convey("second run changes nothing", func() {
				again, err := e.Get(ctx, root, []string{"./..."})
				So(err, ShouldBeNil)
				So(again.Generated, ShouldBeFalse)
				So(again.Written, ShouldEqual, 0)
				So(again.GoFile, ShouldBeFalse)
			})

			Convey("source change regenerates schemas", func() {
				src := filepath.Join(root, "legacy/v1/types.go")
				data, _ := os.ReadFile(src)
				writeFile(t, src, string(data)+"\n// comment\n")
				again, err := e.Get(ctx, root, []string{"./api"})
				So(err, ShouldBeNil)
				So(again.Generated, ShouldBeTrue)
			})

			Convey("prune removes schemas without marks", func() {
				writeFile(t, filepath.Join(root, "api/spec.go"), "package api\n")
				again, err := e.Prune(ctx, []string{"./..."}, nil)
				So(err, ShouldBeNil)
				So(again.Roots, ShouldEqual, 0)
				So(again.Types, ShouldEqual, 0)
				So(again.Removed, ShouldEqual, 1)
			})
		})

		Convey("get of a single package keeps other roots", func() {
			_, err := e.Get(ctx, root, []string{"./..."})
			So(err, ShouldBeNil)
			report, err := e.Get(ctx, filepath.Join(root, "legacy/v1"), []string{"."})
			So(err, ShouldBeNil)
			So(report.Roots, ShouldEqual, 1)
			So(report.Types, ShouldEqual, 3)
		})

		Convey("invalid marks are reported with positions", func() {
			writeFile(t, filepath.Join(root, "api/bad.go"), "package api\n\ntype Bad struct {\n\tN int `crd:\"kubebuild\"`\n}\n")
			_, err := e.Get(ctx, root, []string{"./api"})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "api/bad.go:4")
		})
	})
}
