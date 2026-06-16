# Elasticsearch + Kibana + Filebeat (логи Docker Compose)

Профиль **`elastic`** в корневом [`compose.yml`](../../compose.yml) поднимает:

- **Elasticsearch** — хранение логов (индексы `filebeat-*`).
- **Kibana** — поиск, фильтры, дашборды, **Discover** для ошибок и метрик по логам.
- **Filebeat** — читает **stdout/stderr контейнеров** этого проекта через Docker API и каталог `/var/lib/docker/containers` (стандартный драйвер `json-file`).

Сервисы **neo4j**, **qdrant**, **matrix-bot**, **docs**, **kg-ingestor** и остальные из `compose.yml` автоматически попадают в поток, если они запущены в том же Docker Engine: у них будет поле `container.name`, `container.image.name`, метки Compose (`com.docker.compose.service` и т.д.).

## Имя проекта Compose

В [`compose.yml`](../../compose.yml) задано `name: knowledge`. Если раньше контейнеры создавались под **другим** именем проекта, после обновления compose их лучше пересоздать (`docker compose down` / `up`), чтобы метки логов были единообразными.

## Ограничения

- **Бот, запущенный на хосте** (`go run ./cmd/bot/`), в Elasticsearch **не попадает** — только контейнер `matrix-bot`.
- Стек заточен под **Linux + Docker** с типичными путями. На rootless / Docker Desktop пути к логам могут отличаться — при необходимости поправьте volume в `compose` для `filebeat`.
- `xpack.security.enabled=false` — **только для локальной разработки**. Не выставляйте кластер в интернет без TLS и паролей.

## Запуск

```bash
# из корня репозитория
docker compose --profile elastic up -d elasticsearch kibana filebeat
```

Или цель в Makefile: `make elastic-up`.

- **Elasticsearch API**: `http://127.0.0.1:9200` (порт можно переопределить `ELASTICSEARCH_PORT`).
- **Kibana**: `http://127.0.0.1:5601` (`KIBANA_PORT`).

Подождите, пока Elasticsearch станет **green/yellow** (healthcheck в compose). Дальше проще один раз выполнить **`make elastic-setup`** (см. ниже) — скрипт создаст **Data view** и дашборды. Вручную: Kibana → **Discover** → **Data view** `filebeat-*`.

## Поиск ошибок и фильтрация по сервису

В Kibana (Discover / Lens):

- фильтр по имени контейнера: поле `container.name` (например `knowledge-qdrant-1`, `knowledge-matrix-bot-1`);
- по сервису Compose: `container.labels.com_docker_compose_service` = `qdrant` | `neo4j` | `matrix-bot` | …;
- текст ошибок: поле `message` или `log` (зависит от парсера), полнотекстовый поиск по `error`, `ERR`, `panic` и т.д.

Имя проекта Compose зафиксировано в [`compose.yml`](../../compose.yml) как `name: knowledge`, чтобы метки `com.docker.compose.project` были стабильными.

## Производительность и «профилирование» workflow

Прямого профилирования CPU в Elasticsearch нет: это **логи и время**. Полезно:

- смотреть **временные метки** (`@timestamp`) и коррелировать всплески с нагрузкой;
- строить визуализации по **числу строк логов в минуту** по `container.name`;
- выгружать интервал в **CSV** из Discover для отчётов.

Для глубокого профилирования Go-сервисов дополнительно используйте `pprof` / трассировку в приложении (отдельно от этого стека).

## Дашборды DemoCare (бот, RAG, Ollama, Qdrant)

Бот (`matrix-bot`) и ингестор (`kg-ingestor`) пишут **JSON** в stderr (slog). В [`filebeat.yml`](filebeat.yml) включён **`decode_json_fields`** для строк, начинающихся с `{"time":` — в Elasticsearch появляются поля `knowledge.msg`, `knowledge.level`, `knowledge.event`, `knowledge.latency_ms` и т.д. (удобно для KQL в TSVB).

События, на которых завязаны графики:

| event                  | сервис       | Заметки |
| ---------------------- | ------------ | ------- |
| `bot_query`            | matrix-bot   | RAG-запросы: `latency_ms`, `query_len`, `answer_len`, `ok`, `sender`, `room_id` |
| `ingest_parsed`        | kg-ingestor  | после разбора документов |
| `ingest_graph_written` | kg-ingestor  | запись в Neo4j |
| `ingest_batch_upserted`| kg-ingestor  | батчи в Qdrant, `latency_ms` |
| `ingest_complete`      | kg-ingestor  | итог прогона |

**Ollama** на хосте в индекс не попадает; в дашборде «Ollama» — только строки из бота/ингестора (ошибки `ollama embed probe`, `embed`, и т.п.).

Установка / обновление (идемпотентно, `overwrite=true`):

```bash
# из корня репозитория
make elastic-up        # Elasticsearch + Kibana + Filebeat
make elastic-setup       # data view + визуализации + дашборды
```

Опционально: `KIBANA_URL`, `KIBANA_AUTH=user:password`.

После выполнения в Kibana → **Dashboard**:

| ID | Назначение |
| --- | --- |
| `knowledge-overview` | Общий: логи по контейнерам, ошибки по сервисам, **бот** (QPM, latency p50/p95, ошибки, топ отправителей), ингестор |
| `knowledge-rag` | **RAG**: только панели по `bot_query` (объём, задержки, сбои, пользователи) |
| `knowledge-ollama` | **Ollama / embed**: сбои embed probe и суммарная активность embed/Ollama в логах |
| `knowledge-qdrant` | **Qdrant**: логи контейнера + события приложения (upsert-батчи, сообщения бота про Qdrant) |

Прямые ссылки (порт по умолчанию): [overview](http://127.0.0.1:5601/app/dashboards#/view/knowledge-overview), [rag](http://127.0.0.1:5601/app/dashboards#/view/knowledge-rag), [ollama](http://127.0.0.1:5601/app/dashboards#/view/knowledge-ollama), [qdrant](http://127.0.0.1:5601/app/dashboards#/view/knowledge-qdrant).

Правки визуализаций и дашбордов — в [`setup-kibana.sh`](setup-kibana.sh), затем снова `make elastic-setup`. После смены полей в Filebeat в Kibana может понадобиться **Stack Management → Data views → DemoCare · filebeat → Refresh field list**.

## Опционально: стандартные дашборды Filebeat

После первого успешного запуска можно один раз выполнить (из корня репозитория), когда **Kibana** уже поднялась:

```bash
docker compose --profile elastic run --rm --entrypoint /usr/share/filebeat/filebeat filebeat \
  setup \
  -E setup.kibana.host=http://kibana:5601 \
  -E output.elasticsearch.hosts=http://elasticsearch:9200
```

Затем в Kibana появятся готовые дашборды Filebeat (если версии совпадают).
