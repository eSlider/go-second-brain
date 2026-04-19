---
title: AI
---

# Инструкция для AI-агентов 

Этот файл описывает, как AI-агенты (Cursor, Claude, GPT, локальные subagents) должны работать с репозиторием **Edelweiss**.

## TL;DR для агента

1. Репозиторий — это **база знаний** о Pflegedienst (служба ухода на дому в Германии).
2. Основной язык контента — **русский**. Термины (Verordnung, Pflegegrad, Kasse, Tour, SGB V …) оставляем в оригинальном немецком написании, расшифровка — в [глоссарии](./docs/project/glossary.md).
3. В `docs/*.stt.txt` — сырые STT-транскрипты (ошибки распознавания, немецкие слова записаны русскими буквами). В `docs/reports/*.md` — уже читабельные, структурированные версии.
4. В `docs/project/` — аналитические артефакты (субъекты, кейсы, процессы, оптимизация).
5. Перед любым крупным изменением — читаем [README](./README.md) и [обзор](./docs/project/overview.md).

## Порядок чтения для нового агента

```
README.md
 └── docs/project/overview.md
      └── docs/project/glossary.md
           └── docs/reports/00-summary.md
                ├── docs/project/subjects/README.md
                ├── docs/project/cases/README.md
                └── docs/project/processes/README.md
```

После этого агент имеет достаточный контекст, чтобы:

- отвечать на вопросы о деятельности компании;
- предлагать оптимизации/автоматизации;
- расширять/уточнять документы на основе новых интервью.

## Конвенции

### Язык и стиль

- Основной язык — русский.
- Немецкие термины не переводим насильно: `Verordnung`, `Pflegegrad`, `Fachkraft`, `PDL`, `Krankenkasse`, `Tour`, `Dienstplan`, `Abrechnung`, `Erstgespräch`, `Medikationsplan`.
- Для каждого нового термина — добавляем запись в `docs/project/glossary.md`.

### Форматирование

- Только **Markdown**, без HTML-исключений.
- Каждый документ начинается с `# Заголовок первого уровня`.
- Списки задач/пунктов — через `-`.
- Для процессов — нумерованные списки или mermaid-диаграммы.

### Идентификация кейсов и субъектов

Стабильные короткие ID (таблицы и термины: [глоссарий](./docs/project/glossary.md#id-reference)):

- Субъекты: [SUBJ-PATIENT](./docs/project/subjects/patient.md), [SUBJ-ANGEHORIGER](./docs/project/subjects/angehoeriger.md), [SUBJ-BETREUER](./docs/project/subjects/betreuer.md), [SUBJ-ARZT](./docs/project/subjects/arzt.md), [SUBJ-KRANKENKASSE](./docs/project/subjects/krankenkasse.md), [SUBJ-APOTHEKE](./docs/project/subjects/apotheke.md), [SUBJ-PDL](./docs/project/subjects/pdl.md), [SUBJ-FACHKRAFT](./docs/project/subjects/fachkraft.md), [SUBJ-HELFER](./docs/project/subjects/pflegehelfer.md), [SUBJ-CURASOFT](./docs/project/subjects/curasoft.md), [SUBJ-MOBILE-APP](./docs/project/subjects/mobile-app.md), [SUBJ-ENTLASSMGMT](./docs/project/subjects/entlassmanagement.md), [SUBJ-LEAD-BROKER](./docs/project/subjects/lead-broker.md), [SUBJ-MDK](./docs/project/subjects/mdk.md), [SUBJ-BETRIEBSPRUEFUNG](./docs/project/subjects/betriebspruefung.md), [SUBJ-DSGVO](./docs/project/subjects/datenschutz.md).
- [UC-01](./docs/project/cases/UC-01-intake-new-client.md)…[UC-16](./docs/project/cases/UC-16-angehoerigen-schulung.md) · [кейсы](./docs/project/cases/README.md).
- [PAIN-01](./docs/project/optimization/pain-points.md#pain-01)…[PAIN-48](./docs/project/optimization/pain-points.md#pain-48) · [боли](./docs/project/optimization/pain-points.md).
- [AUTO-01](./docs/project/optimization/automation-opportunities.md#auto-01)…[AUTO-20](./docs/project/optimization/automation-opportunities.md#auto-20) · [автоматизации](./docs/project/optimization/automation-opportunities.md).
- [AGENT-01](./docs/project/optimization/ai-agent-ideas.md#agent-01)…[AGENT-10](./docs/project/optimization/ai-agent-ideas.md#agent-10) · [агенты](./docs/project/optimization/ai-agent-ideas.md).
- [ROAD-01](./docs/project/optimization/roadmap.md#road-01)…[ROAD-08](./docs/project/optimization/roadmap.md#road-08) · [roadmap](./docs/project/optimization/roadmap.md).

Сокращения и немецкие термины — в [глоссарии](./docs/project/glossary.md) (якоря, например `#pdl`, `#sgb-v`, `#verordnung`).

### Работа с docs/*.stt.txt

- НЕ редактируем сырые файлы `*.stt.txt`.
- Любая очистка/реструктуризация — в копии внутри `docs/reports/`.
- При создании нового отчёта:
  1. Указываем источник (`Источник: docs/<имя>.stt.txt`).
  2. Сохраняем смысл, но убираем речевой шум, повторы, оговорки.
  3. Восстанавливаем немецкие термины в корректном написании (например: «феррорунг» → `Verordnung`, «фахкрафт» → `Fachkraft`, «флеги динст» → `Pflegedienst`).
  4. Разбиваем на логические разделы с заголовками.

### Обновление проектных документов

Если из нового материала появляется новая информация:

1. Добавляем её в соответствующий файл `docs/project/subjects/<…>.md`, `docs/project/cases/<…>.md` или `docs/project/processes/<…>.md`.
2. Обновляем `docs/project/glossary.md` при появлении нового термина.
3. Если это новая боль или автоматизация — добавляем в `docs/project/optimization/`.

### Git и минимальные изменения

- Делаем минимальные точечные правки.
- Один PR/коммит — одна смысловая единица (новое интервью, новый кейс, новый субъект).
- Сообщения коммитов — на русском или английском, на усмотрение автора, но стабильно в рамках ветки.

## Подходящие skills

Для этого проекта актуальны:

- **create-rule** — если нужно оформить устойчивую инструкцию для Cursor.
- **create-skill** — если понадобится специализированный skill (например, «конвертер STT-транскрипта в report»).
- **jochen-partner-context** — только как шаблон работы с транскриптами интервью; для Edelweiss не применяется.

Код индексации и Matrix-бота — в [`services/`](./services/); схема стека и контрибьютинг — в [CONTRIBUTING.md](./CONTRIBUTING.md).

## Запрещено

- Менять сырые транскрипты.
- Изобретать факты, не подтверждённые интервью. Если чего-то нет в источниках — помечаем `TODO: уточнить у CTO`.
- Использовать английские/немецкие термины без сверки с `docs/project/glossary.md`.
