# Edelweiss — Проект анализа и автоматизации Pflegedienst

Репозиторий для анализа деятельности немецкого **Pflegedienst** (службы ухода на дому) Edelweiss. Цель — выявить, **что делает компания**, **как** именно, **какие кейсы** существуют, и **что можно оптимизировать/автоматизировать** с помощью ИИ-агентов и интеграций.

## Источники

- Сырой материал — аудио-транскрипты интервью в [`docs/`](./docs).
  - Собеседник: **Artem M.** (CTO, Edelweiss)
  - Интервьюер: **Andrey** (задаёт вопросы, уточняет процессы)
- Обработанный, читаемый формат — в [`docs/reports/`](./docs/reports).

## Структура репозитория

```
edelweiss/
├── README.md                        # Этот файл
├── AGENTS.md                        # Инструкции для AI-агентов / skills
├── docs/
│   ├── *.stt.txt                    # Сырые транскрипты интервью
│   ├── reports/                     # Структурированные версии интервью
│   │   ├── 00-summary.md            # Сквозная сводка всех интервью
│   │   ├── 01-interview-2026-04-17_16-22.md
│   │   └── 02-interview-2026-04-17_16-37.md
│   └── project/                     # Аналитика и проектные артефакты
│       ├── overview.md              # Что делает компания (high-level)
│       ├── company-info.md          # Юр. реквизиты, адрес, IK-Nummer и т.д.
│       ├── glossary.md              # Термины (немецкие + отраслевые)
│       ├── subjects/                # Субъекты (actors) — кто участвует
│       ├── cases/                   # Use cases (кейсы) — что происходит
│       ├── processes/               # End-to-end процессы
│       ├── systems/                 # IT-системы и их роли
│       └── optimization/            # Боли, автоматизация, идеи AI-агентов
└── .cursor/
    └── rules/                       # Правила для Cursor IDE
```

## Как читать проект

1. **Быстрый контекст**
   → [`docs/project/overview.md`](./docs/project/overview.md) — кратко, чем живёт компания.
   → [`docs/project/company-info.md`](./docs/project/company-info.md) — юридические реквизиты Edelweiss.
   → [`docs/project/glossary.md`](./docs/project/glossary.md) — расшифровка терминов (Pflegegrad, Verordnung, SGB V, Tour, PDL, Fachkraft и т.д.); якоря для сокращений — в начале глоссария.
   → [`docs/project/id-reference.md`](./docs/project/id-reference.md) — таблица ссылок `SUBJ-*`, `UC-*`, `PAIN-*`, `AUTO-*`, `AGENT-*`, `ROAD-*`.
2. **Детали интервью**
   → [`docs/reports/00-summary.md`](./docs/reports/00-summary.md).
3. **Структурный анализ**
   → [`docs/project/subjects/`](./docs/project/subjects) — кто (пациент, врач, касса, сотрудник, система).
   → [`docs/project/cases/`](./docs/project/cases) — кейсы и сценарии.
   → [`docs/project/processes/`](./docs/project/processes) — как склеивается end-to-end.
4. **Точки роста**
   → [`docs/project/optimization/`](./docs/project/optimization) — где болит и что можно улучшить.

## Жанр проекта

Это **исследовательская база знаний**, а не код. Цели:

- зафиксировать предметную область (domain knowledge) в виде документов;
- выделить чёткие **субъекты**, **кейсы**, **процессы**;
- сделать материал удобным и для человека, и для **AI-агентов** (чтение, reasoning, генерация решений).

## Работа с AI-агентами

См. [`AGENTS.md`](./AGENTS.md) — там собраны инструкции для skills/агентов: что читать в первую очередь, как обновлять документы, какие конвенции использовать.

## Статус

Живой документ. При появлении новых интервью — кладём `.stt.txt` в `docs/`, после обработки — `*.md` в `docs/reports/`, а обобщения — в `docs/project/`.
