# crd.tools

Инструмент `go tool crd.tools` для пакета [crd.tools/crd](https://github.com/crd-tools/crd):
генерирует манифесты CRD и документ OpenAPI для SDK, собирает схемы чужих типов из
kubebuilder-маркеров.

## Установка

Нужен Go 1.26 или новее.

```sh
go get -tool crd.tools
go tool crd.tools --help
```

Инструмент попадает в `go.mod` директивой `tool` и запускается через
`go tool crd.tools`. Подробнее, в том числе про отдельный `tools.mod`, — в
[docs/tool.md](docs/tool.md).

## Использование

Определения CRD регистрируются в `init` обычного пакета:

```go
package resources

import "crd.tools/crd"

var App = crd.New[apps.App]().
	Group("example.com").Kind("App").Plural("apps").
	Version("v1", true, true)

func init() {
	crd.Add(App)
}
```

Манифесты и список CRD:

```sh
go tool crd.tools generate -o ./config/crd ./...             # файлы <group>_<plural>.yaml
go tool crd.tools generate --validate -o ./config/crd ./...  # с проверкой кодом API-сервера
go tool crd.tools list ./...                                 # таблица найденных CRD
```

OpenAPI для SDK:

```sh
go tool crd.tools openapi prefix add k8s.io.apimachinery.pkg apimachinery
go tool crd.tools openapi generate -o api.yaml ./...
```

Схемы чужих типов из kubebuilder-маркеров:

```sh
go tool crd.tools init                 # один раз: crd.tool.yaml и пакет для снимков
go tool crd.tools kubebuild get ./...  # собрать снимки для полей с меткой crd:"kubebuild"
```

## Документация

| Хочу | Читать |
|---|---|
| установить, обновить, собрать свой инструмент со своим именем | [docs/tool.md](docs/tool.md) |
| генерировать манифесты и смотреть список CRD | [docs/generate.md](docs/generate.md) |
| получить OpenAPI с отдельными типами для SDK | [docs/openapi.md](docs/openapi.md) |
| собрать схемы чужих типов из kubebuilder-маркеров | [docs/kubebuild.md](docs/kubebuild.md) |
| поменять сам инструмент | [docs/development.md](docs/development.md) |

Как описывать схемы и CRD в коде — в документации пакета
[crd.tools/crd](https://github.com/crd-tools/crd).

## Author

[Oleg Shevelev][mantyr]

[mantyr]: https://github.com/mantyr
