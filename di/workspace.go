package di

import (
	"crd.tools/components/workspace"
)

// GetWorkspace возвращает рабочий модуль
func (c *Container) GetWorkspace() workspace.Workspace {
	if c.Workspace == nil {
		return nil
	}
	return c.Workspace
}
