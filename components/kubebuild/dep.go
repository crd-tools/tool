package kubebuild

import (
	"crd.tools/components/workspace"
)

// Dep это зависимости компонента от контейнера
type Dep interface {
	GetToolName() string
	GetWorkspace() workspace.Workspace
}
