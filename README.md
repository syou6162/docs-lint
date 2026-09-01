# docs-lint

roadmap のタスクファイル（`docs/roadmap/**/*.md`）を検査し、依存関係が解消済みのタスクを一覧する CLI。

## インストールと使い方

```console
$ go install github.com/syou6162/docs-lint/cmd/docs-lint@latest
$ docs-lint validate
```

引数を省略すると `docs/roadmap` を検査します。違反を `path: message` の形式で標準出力に並べ、1 件以上あれば exit 1。

```
usage: docs-lint <subcommand> [args]

subcommands:
  validate [dir]  validate roadmap task files
  tasks [dir]     list available roadmap tasks
```

`tasks` サブコマンドは、依存が解消済みのタスクだけを一覧します。`--priority`、`--type`、`--sort`、`--json` で絞り込み・並び順・出力形式を指定できます。

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
- run: go install github.com/syou6162/docs-lint/cmd/docs-lint@latest
- run: docs-lint validate
```
