package di

import (
	"errors"

	"crd.tools/components/kubebuild"
	"crd.tools/components/workspace"
)

// Container это контейнер компонентов инструмента
type Container struct {
	// ToolName имя инструмента для сообщений
	ToolName string

	Workspace *workspace.Component
	Kubebuild *kubebuild.Component
}

// New возвращает контейнер со всеми компонентами
func New(toolName string) (*Container, error) {
	if toolName == "" {
		return nil, errors.New("empty tool name")
	}
	c := &Container{ToolName: toolName}
	c.Workspace = workspace.New(c)
	c.Kubebuild = kubebuild.New(c)
	return c, nil
}

// GetToolName возвращает имя инструмента
func (c *Container) GetToolName() string {
	return c.ToolName
}
