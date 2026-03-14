# slack-router

Slack の Slash Command を Socket Mode で受け取り、設定ファイルのルーティングテーブルに従ってローカルのシェルスクリプトをサブプロセスとして非同期実行するデーモンです。

システム管理・LLM 連携・デプロイ自動化などの ChatOps を安全かつスケーラブルに実現するハブとして機能します。

---

## 機能

- **Socket Mode 接続** — インバウンドのポート開放不要でイベントを受信
- **コマンドルーティング** — YAML 設定ファイルで Slash Command とスクリプトを紐づけ
- **安全なパラメータ渡し** — コマンドメタデータを `stdin` 経由の JSON で渡す（argv を使用しないため `ps aux` からの情報漏洩を防止）
- **DoS 対策** — グローバルおよびコマンド単位の同時実行数上限（セマフォ）
- **タイムアウト強制停止** — SIGTERM → SIGKILL でプロセスツリーごと終了
- **構造化ログ** — JSON 形式でルーターとワーカーの動作を記録

---

## 必要要件

- Go 1.22 以上
- Slack App（Socket Mode 有効・Slash Command 登録済み）
  → 設定手順は [docs/slack-setup.md](docs/slack-setup.md) を参照

---

## インストール

```bash
git clone https://github.com/magifd2/slack-router.git
cd slack-event-handler
make build
```

バイナリ (`./slack-router`) が生成されます。依存関係はバイナリに同梱されるため、配置先サーバーに Go 環境は不要です。

---

## 設定

`config.example.yaml` をコピーして編集します。

```bash
cp config.example.yaml config.yaml
cp .env.example .env  # トークンはここに記入
```

**トークンは環境変数で渡すことを推奨します。** `config.yaml` をリポジトリに含めても安全になります。

```bash
# .env（Git 管理対象外）
SLACK_APP_TOKEN=xapp-1-...
SLACK_BOT_TOKEN=xoxb-...
```

起動時に環境変数を読み込む例:

```bash
# source して起動
set -a && source .env && set +a
./slack-router -config config.yaml

# または env コマンドで渡す
env $(cat .env | xargs) ./slack-router -config config.yaml
```

`config.yaml` に直接書くこともできますが、その場合は `.gitignore` に追加してリポジトリに含めないでください。

```yaml
slack:
  app_token: "xapp-1-..."  # 非推奨: 直書きはローカル開発のみ
  bot_token:  "xoxb-..."
```

環境変数が設定されている場合は常に `config.yaml` の値より優先されます。

### 設定項目

| キー | 必須 | デフォルト | 説明 |
|---|---|---|---|
| `slack.app_token` / `SLACK_APP_TOKEN` | ✓ | — | `xapp-` から始まる App-Level Token |
| `slack.bot_token` / `SLACK_BOT_TOKEN` | ✓ | — | `xoxb-` から始まる Bot Token |
| `global.max_concurrent_workers` | | `10` | 全コマンド合計の同時実行上限 |
| `global.log_level` | | `info` | `debug` / `info` / `warn` / `error` |
| `routes[].command` | ✓ | — | Slash Command 名（例: `/ask`） |
| `routes[].script` | ✓ | — | 実行するスクリプトのパス |
| `routes[].timeout` | | `5m` | タイムアウト（Go の duration 形式: `30s`, `5m`, `1h`） |
| `routes[].max_concurrency` | | 無制限 | このコマンドの同時実行上限 |
| `routes[].allow_channels` | | 無制限 | 実行を許可するチャンネル ID のリスト |
| `routes[].allow_users` | | 無制限 | 実行を許可するユーザー ID のリスト |
| `routes[].deny_channels` | | なし | 実行を拒否するチャンネル ID のリスト |
| `routes[].deny_users` | | なし | 実行を拒否するユーザー ID のリスト |

### アクセス制御 (ACL)

各ルートに `allow_channels` / `allow_users` / `deny_channels` / `deny_users` を設定することで、コマンドの実行権限をチャンネル・ユーザー単位で制御できます。

```yaml
routes:
  - command: "/deploy"
    script: "./scripts/deploy.sh"
    allow_channels:
      - "C000000001"  # #ops のみ
    allow_users:
      - "U000000001"  # @alice
      - "U000000002"  # @bob
    deny_users:
      - "U000000099"  # 一時的に制限
```

**評価順序（優先度高い順）:**

| 順序 | ルール | 空の場合 |
|---|---|---|
| 1 | `deny_users` | スキップ |
| 2 | `deny_channels` | スキップ |
| 3 | `allow_users` | 全ユーザー許可 |
| 4 | `allow_channels` | 全チャンネル許可 |

- deny は allow より常に優先されます
- allow リストが空（未設定）の場合は「全員許可」として扱います
- 拒否された場合、ユーザーには「権限がありません」とだけ通知されます（どのルールにマッチしたかは伏せます）

チャンネル ID / ユーザー ID は Slack の URL やプロフィール画面から確認できます（`C` から始まるのがチャンネル、`U` から始まるのがユーザー）。

---

## ワーカースクリプトの書き方

ルーターはスクリプトを起動し、以下の JSON を `stdin` に書き込んで閉じます。スクリプトは `stdin` を読み取り、必要な処理を行います。

動作確認用のサンプルスクリプトとして [`scripts/hello.sh`](scripts/hello.sh) を用意しています。`/hello` コマンドで呼び出すと、実行者に挨拶を返します。

### stdin に流れてくる JSON

```json
{
  "command":      "/ask",
  "text":         "こんにちは",
  "user_id":      "U123456",
  "channel_id":   "C123456",
  "response_url": "https://hooks.slack.com/commands/..."
}
```

### Bash スクリプトの例

```bash
#!/usr/bin/env bash
set -euo pipefail

# stdin から JSON を読み取る
payload=$(cat)

command=$(echo "$payload"     | jq -r '.command')
text=$(echo "$payload"        | jq -r '.text')
user_id=$(echo "$payload"     | jq -r '.user_id')
response_url=$(echo "$payload" | jq -r '.response_url')

# 処理（例: Slack へ返信）
curl -s -X POST "$response_url" \
  -H "Content-Type: application/json" \
  -d "{\"text\": \"<@${user_id}> あなたのメッセージ: ${text}\"}"
```

### ルーターとワーカーの責務分担

```
[Slack] ──slash command──▶ [slack-router]
                               │
                               ├─ ACK（3秒以内）
                               ├─ DoS チェック
                               └─ script 起動 ──stdin JSON──▶ [worker script]
                                                                    │
                                                                    └─ response_url POST ──▶ [Slack]
```

ルーターは Slack への「返信」に関与しません。返信はワーカースクリプトが `response_url` を通じて行います（疎結合）。

---

## 起動

```bash
./slack-router -config config.yaml
```

| フラグ | デフォルト | 説明 |
|---|---|---|
| `-config` | `config.yaml` | 設定ファイルのパス |

### ログ出力例

```json
{"time":"2026-03-14T10:00:00Z","level":"INFO","msg":"slack-router starting","routes":2,"max_concurrent_workers":10}
{"time":"2026-03-14T10:00:01Z","level":"INFO","msg":"connected to slack"}
{"time":"2026-03-14T10:01:00Z","level":"INFO","msg":"slash command received","command":"/ask","text":"こんにちは","user":"U123456","channel":"C123456"}
{"time":"2026-03-14T10:01:00Z","level":"INFO","msg":"worker started","pid":12345,"command":"/ask","script":"./scripts/ask_llm.sh","user":"U123456"}
{"time":"2026-03-14T10:01:02Z","level":"INFO","msg":"worker exited normally","pid":12345,"command":"/ask"}
```

---

## グレースフルシャットダウン

`SIGINT`（Ctrl+C）または `SIGTERM` を受け取ると、新規リクエストの受付を停止し、実行中のワーカープロセスがすべて終了するまで待機してから終了します。

---

## Makefile

```bash
make build   # バイナリをビルド
make run     # ビルドして起動（config.yaml を使用）
make tidy    # go mod tidy
make lint    # go vet
```

---

## アーキテクチャ

```
main.go       — Socket Mode イベントループ・シグナルハンドリング
config.go     — YAML 設定の読み込みとバリデーション
router.go     — コマンドルーティング・グローバル/ルート別セマフォ管理
worker.go     — exec.Command・stdin JSON 注入・タイムアウト処理
```

### セマフォの構造

```
[global semaphore]  max_concurrent_workers = 10
    └── [/ask semaphore]    max_concurrency = 3
    └── [/deploy semaphore] max_concurrency = 1
```

上限に達したリクエストは即座にドロップされ、`response_url` 経由でユーザーに通知されます。

---

## ライセンス

MIT
