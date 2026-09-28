# Документ OpenAPI для SDK

Как получить документ OpenAPI v3 по определениям CRD проекта. Зачем он нужен,
как строятся имена компонентов, как работают замены префиксов и ссылки — в
документации пакета ([openapi.md](https://github.com/crd-tools/crd/blob/main/docs/openapi.md)).

## Пример

```sh
go tool crd.tools openapi prefix add k8s.io.apimachinery.pkg apimachinery
go tool crd.tools openapi prefix add example.com.project
go tool crd.tools openapi generate -o api.yaml ./...
```

```yaml
openapi: 3.0.3
components:
  schemas:
    workers.Worker:
      properties:
        spec: {$ref: '#/components/schemas/workers.WorkerSpec'}
        status: {$ref: '#/components/schemas/workers.WorkerStatus'}
    workers.WorkerSpec:
      properties:
        replicas: {type: integer, format: int32, minimum: 0, maximum: 100}
        selector: {$ref: '#/components/schemas/apimachinery.apis.meta.v1.LabelSelector'}
    apimachinery.apis.meta.v1.LabelSelector:
      properties:
        matchExpressions:
          items: {$ref: '#/components/schemas/apimachinery.apis.meta.v1.LabelSelectorRequirement'}
```

## Замены префиксов

Замены хранятся в `crd.openapi.yaml` в корне модуля:

```yaml
version: 1
info:
  title: workers
  version: v1
prefixes:
  - from: example.com.project
    to: ""
  - from: k8s.io.apimachinery.pkg
    to: apimachinery
```

```sh
go tool crd.tools openapi prefix add <from> [to]   # добавить или изменить, без to — отрезать
go tool crd.tools openapi prefix remove <from>
go tool crd.tools openapi prefix list
```

`info` необязателен: по умолчанию заголовок — путь модуля, версия — `v1`.

## Генерация

```sh
go tool crd.tools openapi generate ./...                    # YAML в stdout
go tool crd.tools openapi generate --format json ./...
go tool crd.tools openapi generate --kind Worker -o api.yaml ./...
```

Определения находятся так же, как в `generate` ([generate.md](generate.md)):
`crd.Add` в `init`, те же фильтры `--kind`, `--group`, `--plural`. Флаги ставятся
до шаблонов пакетов.

## Смотрите также

- [openapi.md](https://github.com/crd-tools/crd/blob/main/docs/openapi.md) — имена компонентов, замены, ссылки;
- [generate.md](generate.md) — как находятся определения.
