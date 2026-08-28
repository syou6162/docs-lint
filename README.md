# docs-lint

Markdown の YAML front-matter を、設定ファイルに書いたルールで検査する CLI。

エージェントに読ませるドキュメント（roadmap のタスクファイル、要件、決定ログなど）を複数リポジトリで運用すると、同じような検査コードが各リポジトリに増える。docs-lint はその検査本体だけを持ち、各リポジトリはルール定義（`docs-lint.yaml`）と CI からの呼び出しだけを置く。

## 使い方

```console
$ go run github.com/syou6162/docs-lint/cmd/docs-lint@latest
docs/roadmap/backlog/slim-slack-server.md: missing required field "priority"
docs/roadmap/backlog/typed-errors.md: depends_on references missing id "slack-auth"
```

```
usage: docs-lint [flags] [dir]

flags:
  -config string
        path to the config file (default: docs-lint.yaml)
```

- `dir` は検査対象のルート（省略時はカレントディレクトリ）。配下の `.md`（拡張子の大小文字は区別しない）を再帰的に見る。`.git` は辿らず、ディレクトリの symlink も辿らない
- `-config` を省略した場合は `docs-lint.yaml` / `docs-lint.yml` / `.docs-lint.yaml` / `.docs-lint.yml` をこの順で探す。設定の探索はカレントディレクトリ基準、検査対象は `dir` 基準なので、別ディレクトリを検査するときは `-config` を明示する
- 違反を `path: message` の形式で標準出力に並べる
- exit code は、違反があれば 1、設定エラー・使い方の誤り・IO エラーなら 2、問題なしなら 0
- front-matter の開始・終了 fence は行全体が `---` である必要がある（本文中の `---` で終了扱いにしない）
- 1 ファイルが複数のルールに一致する場合、各ルールが独立に適用される。`allow_unknown_fields: false` のルールが 2 つ一致すると互いのフィールドを未知と報告するので、`include` / `exclude` は重複しないように書く

## 設定

```yaml
rules:
  - name: roadmap-task
    include:
      - "docs/roadmap/**/*.md"
    exclude:
      - "**/AGENTS.md"
      - "**/overview.md"
    filename_field: id
    fields:
      id:
        type: string
        required: true
        pattern: "^[a-z0-9]+(-[a-z0-9]+)*$"
        unique: true
      title:
        type: string
        required: true
      type:
        type: string
        required: true
        enum: [bug, refactoring, documentation, test, feature]
      priority:
        type: string
        required: true
        enum: [high, medium, low]
      depends_on:
        type: string_array
        required: true
        references: id
        acyclic: true
```

### rule

| キー | 既定値 | 説明 |
|---|---|---|
| `name` | 必須 | ルール名。エラー文の識別に使う。重複不可 |
| `include` | 必須 | 検査対象のパターン。`**` を含む glob をルートからの相対パスに対して照合する |
| `exclude` | `[]` | `include` から除外するパターン |
| `filename_field` | なし | ファイル名（`.md` を除く）が一致していなければならないフィールド名 |
| `allow_unknown_fields` | `false` | `fields` に無い front-matter を許すかどうか |
| `fields` | 必須 | フィールド名 → 検査内容 |

### field

| キー | 既定値 | 説明 |
|---|---|---|
| `type` | 必須 | `string`（非空文字列）または `string_array`（非空文字列の配列） |
| `required` | `false` | 未定義ならエラーにする |
| `pattern` | なし | 正規表現。`string_array` では各要素に適用する |
| `enum` | なし | 許可する値（`string` のみ） |
| `unique` | `false` | 同じルールに一致する全ファイル間で値が一意（`string` のみ） |
| `references` | なし | 各要素が、同じルール内の指定フィールドの値として実在すること（`string_array` のみ） |
| `self_reference_allowed` | `false` | `references` で自分自身の値を参照してよいか |
| `acyclic` | `false` | `references` の参照グラフに閉路が無いこと |

設定ファイルは未知のキーを拒否し、glob と正規表現は読み込み時に検証する。ルール定義の綴り間違いで検査が黙って無効になるのを防ぐため。

## CI

```yaml
- uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0
  with:
    go-version: stable
- run: go run github.com/syou6162/docs-lint/cmd/docs-lint@latest
```

## 設計上の判断

- **ルールを設定で書く**。Go コードで書けるようにすると、各リポジトリが再び Go の依存とテストを抱えることになるため、呼び出し側は YAML だけで済むようにしている
- **front-matter に限定しない名前にしている**。本文の必須見出しやファイル名規約の検査を足しても名前が嘘にならないようにするため
- **本文の検査はまだ持っていない**。front-matter の検査だけで既存の運用（roadmap のタスクファイル）を置き換えられるため、必要になってから足す
- **docs-lint 自身も docs-lint で検査する**。リポジトリ root の `docs-lint.yaml` が `docs/decisions/` を対象にしており、CI で `go run ./cmd/docs-lint` を実行している
