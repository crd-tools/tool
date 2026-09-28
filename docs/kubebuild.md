# Снимки из kubebuilder-маркеров

Как собрать снимки схем типов, описанных маркерами `// +kubebuilder:...`. Что
такое метка `crd:"kubebuild"`, как снимки подключаются в приложении и как
устроен их формат — в документации пакета
([kubebuild.md](https://github.com/crd-tools/crd/blob/main/docs/kubebuild.md)).

Инструмент находит поля с меткой, прогоняет их типы через
[controller-tools](https://github.com/kubernetes-sigs/controller-tools) и
кладёт готовые схемы в пакет вашего проекта. Маркеры читаются только у типов, до
которых инструмент дошёл через метку.

## Пример

```go
type Spec struct {
	Resources   legacy.Resources    `json:"resources" crd:"kubebuild"`
	Tolerations []corev1.Toleration `json:"tolerations" crd:"kubebuild"`
}
```

```sh
go tool crd.tools init                  # один раз: crd.tool.yaml и пакет для снимков
go tool crd.tools kubebuild get ./...   # собрать снимки
```

```go
import "example.com/project/internal/tool/crd/schemas"

func init() {
	crd.SetDERSource(schemas.FS())
}
```

Дальше поля с меткой получают схему из снимков в `SchemaFor` и при сборке CRD.

## Метка

Метку в инструменте меняют настройки собственного инструмента
([tool.md](tool.md)); приложению её нужно указать через `crd.SetKubebuildTag`.

Метку можно ставить и на встроенное поле: так подключается чужой CRD целиком
([kubebuild.md](https://github.com/crd-tools/crd/blob/main/docs/kubebuild.md#чужой-crd-целиком)).
`kubebuild get` соберёт снимок встроенного типа так же, как для обычного поля.

Снимки хранят схемы с исходными тегами проекта, например каналами Gateway API.
Теги применяют модификаторы определения
([kubebuild.md](https://github.com/crd-tools/crd/blob/main/docs/kubebuild.md#модификаторы)),
поэтому из одного снимка получается любой канал. Модификаторы задаются только в
коде: `generate`, `list` и `openapi generate` собирают определения так, как они
описаны, и не переопределяют их.

На поля, тип которых не разворачивается до именованного (встроенные типы,
дженерики, анонимные структуры, интерфейсы), инструмент выдаёт ошибку с позицией
в коде.

## Команды

```sh
go tool crd.tools init                          # crd.tool.yaml, пакет internal/tool/crd/schemas
go tool crd.tools init ./pkg/crdschemas         # свой каталог пакета со снимками

go tool crd.tools kubebuild get                 # текущий пакет (.)
go tool crd.tools kubebuild get ./...           # весь модуль или любые шаблоны go list
go tool crd.tools kubebuild get github.com/x/y  # внешний пакет, запоминается в настройках

go tool crd.tools kubebuild prune               # пересобрать по всему проекту и удалить лишнее
```

- `init` создаёт файл настроек в корне модуля. Повторный `init` — ошибка.
- `kubebuild get` дособирает снимки для помеченных полей и **ничего не удаляет**,
  поэтому его можно запускать из любого подкаталога.
- `kubebuild prune` всегда обходит весь проект, независимо от текущего каталога,
  и удаляет снимки, на которые больше нет меток.

Если снимки свежие, controller-tools не запускается, и повторный вызов занимает
доли секунды. Свежесть проверяется по версии модуля из `go.mod` проекта (для
пакетов своего модуля — по хэшу исходников) и по версии инструмента.

## Файл настроек

`crd.tool.yaml` в корне модуля:

```yaml
version: 1
output: internal/tool/crd/schemas
kubebuild:
  scan:
    - ./...
  external:
    - github.com/mantyr/starter
```

- `output` — каталог пакета со снимками относительно корня модуля;
- `kubebuild.scan` — шаблоны пакетов модуля, которые обходит `prune`;
- `kubebuild.external` — внешние пакеты из `get`; `prune` обходит их вместе со
  `scan`, иначе удалил бы их снимки.

## Проверка в CI

Вывод детерминирован: одинаковые исходники дают одинаковые байты.

```sh
go tool crd.tools kubebuild prune
git diff --exit-code internal/tool/crd/schemas
```

## Смотрите также

- [kubebuild.md](https://github.com/crd-tools/crd/blob/main/docs/kubebuild.md) — метка, подключение и формат снимков;
- [tool.md](tool.md) — установка и настройка инструмента.
