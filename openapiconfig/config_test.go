package openapiconfig

import (
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfig(t *testing.T) {
	Convey("Config", t, func() {
		file := filepath.Join(t.TempDir(), "crd.openapi.yaml")

		c, err := Load(file)
		So(err, ShouldBeNil)
		So(c.Prefixes, ShouldBeEmpty)

		So(c.Add("k8s.io.apimachinery.pkg", "apimachinery"), ShouldBeNil)
		So(c.Add("example.com.project", ""), ShouldBeNil)
		So(c.Add("k8s.io.apimachinery.pkg", "am"), ShouldBeNil) // замена существующего
		So(c.Add("", "x"), ShouldNotBeNil)
		So(c.Save(file), ShouldBeNil)

		again, err := Load(file)
		So(err, ShouldBeNil)
		So(again.Prefixes, ShouldHaveLength, 2)
		So(again.Prefixes[0].From, ShouldEqual, "example.com.project")
		So(again.Prefixes[1].To, ShouldEqual, "am")

		So(again.Remove("example.com.project"), ShouldBeTrue)
		So(again.Remove("nope"), ShouldBeFalse)
		So(again.Prefixes, ShouldHaveLength, 1)
	})
}
