package list

import (
	"bytes"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"crd.tools/generate"
)

var rows = []generate.Row{
	{Name: "apps.example.com", Kind: "App", Group: "example.com", Plural: "apps", Scope: "Namespaced",
		Versions: []string{"v1", "v1beta1"}, Storage: "v1"},
	{Kind: "NoGroup", Plural: "nogroups", Error: "crd: empty group"},
}

func render(value string) (string, int, error) {
	var buf bytes.Buffer
	failed, err := printRows(&buf, value, rows)
	return buf.String(), failed, err
}

func TestPrintRows(t *testing.T) {
	Convey("printRows", t, func() {
		Convey("table by default: header, dashes, storage marked", func() {
			out, failed, err := render("table")
			So(err, ShouldBeNil)
			So(failed, ShouldEqual, 1)
			lines := strings.Split(strings.TrimSpace(out), "\n")
			So(lines, ShouldHaveLength, 3)
			So(strings.Fields(lines[0]), ShouldResemble, []string{"NAME", "KIND", "GROUP", "PLURAL", "SCOPE", "VERSIONS", "STATUS"})
			So(lines[1], ShouldContainSubstring, "v1*,v1beta1")
			So(strings.Fields(lines[2])[0], ShouldEqual, "-")
			So(lines[2], ShouldContainSubstring, "error: crd: empty group")
		})

		Convey("json: one object per line, error only when present", func() {
			out, _, err := render("json")
			So(err, ShouldBeNil)
			lines := strings.Split(strings.TrimSpace(out), "\n")
			So(lines, ShouldHaveLength, 2)
			So(lines[0], ShouldNotContainSubstring, `"error"`)
			So(lines[1], ShouldContainSubstring, `"error":"crd: empty group"`)
			So(lines[0], ShouldContainSubstring, `"versions":"v1,v1beta1"`)
		})

		Convey("yaml: documents are separated", func() {
			out, _, err := render("yaml")
			So(err, ShouldBeNil)
			So(strings.Count(out, "---\n"), ShouldEqual, 2)
		})

		Convey("custom template", func() {
			out, _, err := render(`{{.Kind}}={{.Storage}}`)
			So(err, ShouldBeNil)
			So(out, ShouldEqual, "App=v1\nNoGroup=\n")
		})

		Convey("unknown field is reported", func() {
			_, _, err := render(`{{.Nmae}}`)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "--format")
		})
	})
}
