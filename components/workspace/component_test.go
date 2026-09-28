package workspace

import (
	"testing"

	"github.com/mantyr/starter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestComponent(t *testing.T) {
	Convey("Checking compatibility", t, func() {
		component := New(nil)
		So(component, ShouldNotBeNil)
		testComponent(component)
		testWorkspace(component)
	})
}

func testComponent(c starter.Component) {}

func testWorkspace(w Workspace) {}
