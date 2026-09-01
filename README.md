# docs-lint

roadmap のタスクファイル（`docs/roadmap/**/*.md`）の YAML front-matter を検査する CLI。

`syou6162/times-agent-talk` の `roadmap validate` をそのまま切り出したもの。

## 使い方

```console
$ go run github.com/syou6162/docs-lint/cmd/docs-lint@latest
docs/roadmap/backlog/slim-slack-server.md: parse: missing required field "priority"
docs/roadmap/backlog/typed-errors.md: validate: depends_on references missing task id "slack-auth"
```

```
usage: docs-lint [dir]

validate roadmap task files (default dir: docs/roadmap)
```

違反を `path: message` の形式で標準出力に並べ、1 件以上あれば exit 1。

## 検査内容

タスクファイルは以下の front-matter を持つ。`AGENTS.md` と `overview.md` は対象外。

```yaml
---
id: slim-slack-server
title: Slack server を薄くする
type: refactoring
priority: high
depends_on: [typed-errors]
---
```

- 必須フィールドは `id` / `title` / `type` / `priority` / `depends_on`。これ以外のフィールドはエラー
- `id` は `^[a-z0-9]+(-[a-z0-9]+)*$` に一致し、ファイル名（`.md` を除く）と一致すること。全ファイルで一意であること
- `type` は `bug` / `refactoring` / `documentation` / `test` / `feature` のいずれか
- `priority` は `high` / `medium` / `low` のいずれか
- `depends_on` は id 文字列の配列。参照先が実在し、自分自身を参照せず、依存関係に閉路が無いこと

## CI

```yaml
- uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0
  with:
    go-version: stable
- run: go run github.com/syou6162/docs-lint/cmd/docs-lint@latest
```
