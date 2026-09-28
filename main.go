package main

import (
	"os"

	"crd.tools/tool"
)

func main() {
	tool.New(tool.Config{
		Name:        "go tool crd.tools",
		Env:         "CRD_TOOLS",
		Usage:       "инструменты для работы с CRD",
		ConfigFile:  "crd.tool.yaml",
		Output:      "internal/tool/crd/schemas",
		OpenAPIFile: "crd.openapi.yaml",
		Kubebuild: tool.Kubebuild{
			TagKey:   "crd",
			TagValue: "kubebuild",
		},
	}).RunAndFatal(os.Args)
}
