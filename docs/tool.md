# Инструмент crd

Установка, обновление, команды и собственный нейминг инструмента. Читайте один
раз при подключении к проекту.

## Установка

Нужен Go 1.26 или новее: этого требует controller-tools v0.22.

```sh
go get -tool crd.tools
go tool crd.tools --help
```

Инструмент попадает в `go.mod` директивой `tool` и запускается через
`go tool crd.tools`. Бинарник собирается в кэш сборки Go, в `GOBIN` ничего не ставится.
Устанавливать через `go install` не нужно: `GOBIN` один на пользователя, и версии
разных проектов перетрут друг друга.

Обновить только этот инструмент:

```sh
go get -tool crd.tools@latest
```

`go get tool` без флага обновляет **все** инструменты модуля.

После обновления инструмента может понадобиться `go mod tidy`: у новой версии
бывают новые зависимости.

## Общий go.mod или отдельный tools.mod

По умолчанию инструмент живёт в `go.mod` проекта, и у них общий граф
зависимостей: для `k8s.io/*` выбирается максимальная из требуемых версий. Поэтому
обновление инструмента может поднять версии `k8s.io/*` в проекте, и наоборот.

Если это нежелательно, объявите инструмент в отдельном файле модуля:

```sh
go get -tool -modfile=tools.mod crd.tools
go tool -modfile=tools.mod crd.tools kubebuild get ./...
```

`-modfile` нужен при каждом вызове, удобно спрятать его в `Makefile`. Пакеты
проекта инструмент в обоих случаях загружает по `go.mod` проекта.

## Команды

| Команда | Для чего | Документ |
|---|---|---|
| `init` | создать `crd.tool.yaml` | [kubebuild.md](kubebuild.md) |
| `kubebuild get`, `kubebuild prune` | снимки из kubebuilder-маркеров | [kubebuild.md](kubebuild.md) |
| `generate` | манифесты CRD | [generate.md](generate.md) |
| `list` | список найденных CRD | [generate.md](generate.md) |
| `openapi generate`, `openapi prefix` | OpenAPI для SDK | [openapi.md](openapi.md) |

- Флаг `-v` у любой команды показывает ход инициализации компонентов.
- Флаги ставятся до шаблонов пакетов: `get -v ./...`.

## Свой инструмент со своим именем

Имя команды для `go tool` — последний элемент пути `main`-пакета, алиасов у
директивы `tool` нет. Поэтому этот инструмент вызывается как `go tool crd.tools`.
Чтобы назвать инструмент иначе или поменять настройки по умолчанию, создайте
свой модуль с одним `main.go`:

```go
// github.com/you/schemas-tool/cmd/sch/main.go
package main

import (
	"os"

	"crd.tools/tool"
)

func main() {
	tool.New(tool.Config{
		Name:        "go tool sch",
		Env:         "SCH",
		Usage:       "схемы kubebuilder-типов",
		ConfigFile:  "sch.tool.yaml",
		Output:      "internal/schemas",
		OpenAPIFile: "sch.openapi.yaml",
		Kubebuild: tool.Kubebuild{
			TagKey:   "schema",
			TagValue: "k8s",
		},
	}).RunAndFatal(os.Args)
}
```

```sh
go get -tool github.com/you/schemas-tool/cmd/sch
go tool sch init
```

Если меняете метку, укажите её и приложению: `crd.SetKubebuildTag("schema", "k8s")`.

| Поле `tool.Config` | Назначение |
|---|---|
| `Name` | команда вызова в справке, подсказках и сообщениях: `go tool sch` |
| `Env` | префикс переменных окружения: `SCH` |
| `Usage` | описание в справке |
| `ConfigFile` | имя файла настроек проекта в корне модуля |
| `Output` | каталог пакета со снимками по умолчанию для `init` |
| `OpenAPIFile` | имя файла настроек OpenAPI в корне модуля |
| `Kubebuild.TagKey`, `Kubebuild.TagValue` | метка в тегах полей |

Значения по умолчанию переопределяются флагами или переменными окружения с
префиксом из `Env`. У `go tool crd.tools` префикс `CRD_TOOLS`:

| Флаг | Переменная |
|---|---|
| `--tool.config` | `CRD_TOOLS_CONFIG` |
| `--openapi.config` | `CRD_TOOLS_OPENAPI_CONFIG` |
| `--kubebuild.tag.key` | `CRD_TOOLS_KUBEBUILD_TAG_KEY` |
| `--kubebuild.tag.value` | `CRD_TOOLS_KUBEBUILD_TAG_VALUE` |
