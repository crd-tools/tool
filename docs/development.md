# Устройство и тесты

Для тех, кто меняет сам инструмент.

## Пакеты

```
main.go               main инструмента: go get -tool crd.tools
tool/                 сборка инструмента: Config, New, RunAndFatal
commands/             команды на github.com/mantyr/app
components/           компоненты на github.com/mantyr/starter: workspace, kubebuild
di/                   контейнер компонентов
kubebuild/            снимки: поиск меток, controller-tools, запись каталога
generate/             generate, list и openapi: поиск crd.Add, временный main, фильтры
openapiconfig/        файл настроек OpenAPI: замены префиксов
project/              файл настроек проекта
gocmd/                вызовы go env и go list
logger/               логгер команд
```

Реестр схем и формат DER живут в пакете `crd.tools/crd/registry`.

`main` лежит в корне модуля, поэтому команда называется `go tool crd.tools`.
Свой инструмент с другим именем собирается на пакете `crd.tools/tool`
([tool.md](tool.md)).

## Локальная разработка вместе с crd.tools/crd

В корне лежит `go.work`, который подключает соседний каталог `../crd`:

```
crd-tools/
├── crd/     # crd.tools/crd
└── tool/    # crd.tools, go.work здесь
```

Пока `crd.tools/crd` не опубликован, в `go.mod` его нет: зависимость берётся
только из `go.work`. После публикации:

```sh
GOWORK=off go get crd.tools/crd@vX.Y.Z
GOWORK=off go mod tidy
```

## Тесты

```sh
go test -race ./...
```

- `kubebuild/engine_test.go` — снимки на временном модуле, использует тулчейн Go,
  пропускается с `-short`.
