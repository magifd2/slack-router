# Slack App セットアップガイド

slack-router を動かすには、Slack App の作成とトークンの取得が必要です。このドキュメントでは Slack の管理画面での手順をステップごとに説明します。

---

## 前提

- Slack ワークスペースの管理者権限、またはアプリのインストール権限
- slack-router をインストールするサーバー（または開発環境）

---

## Step 1: Slack App を作成する

1. [Slack API: Your Apps](https://api.slack.com/apps) にアクセスし、**Create New App** をクリック
2. **From scratch** を選択
3. **App Name** に任意の名前を入力（例: `slack-router`）
4. インストール先の **Workspace** を選択
5. **Create App** をクリック

---

## Step 2: Socket Mode を有効化する

1. 左サイドバーの **Settings > Socket Mode** をクリック
2. **Enable Socket Mode** をオン
3. **Token Name** に任意の名前を入力（例: `socket-mode-token`）
4. **Generate** をクリック
5. 表示された `xapp-1-...` のトークンをコピーして保管する

> このトークンが `config.yaml` の `slack.app_token` に入ります。

---

## Step 3: Bot Token スコープを設定する

1. 左サイドバーの **Features > OAuth & Permissions** をクリック
2. **Scopes > Bot Token Scopes** セクションで **Add an OAuth Scope** をクリック
3. 以下のスコープを追加する

| スコープ | 用途 |
|---|---|
| `commands` | Slash Command の受信 |

> ワーカースクリプトから Slack API（メッセージ投稿など）を直接呼び出す場合は、用途に応じて `chat:write` などを追加してください。ただし slack-router 本体は Slack API を直接呼びません。

---

## Step 4: ワークスペースにインストールする

1. 左サイドバーの **Settings > Install App** をクリック
2. **Install to Workspace** をクリック
3. 権限の確認画面で **許可する** をクリック
4. インストール完了後、**Bot User OAuth Token**（`xoxb-...`）をコピーして保管する

> このトークンが `config.yaml` の `slack.bot_token` に入ります。

---

## Step 5: Slash Command を登録する

ルーティングしたいコマンド分だけ以下の手順を繰り返します。

1. 左サイドバーの **Features > Slash Commands** をクリック
2. **Create New Command** をクリック
3. 以下を入力する

| 項目 | 入力例 | 説明 |
|---|---|---|
| **Command** | `/ask` | スラッシュコマンド名 |
| **Request URL** | `https://example.com/` | Socket Mode では使用されないため任意の URL で可 |
| **Short Description** | LLM に質問する | Slack 上に表示される説明文 |
| **Usage Hint** | `[質問内容]` | `/ask` の後ろに続くパラメータのヒント（任意） |

4. **Save** をクリック

> `config.yaml` の `routes[].command` と完全一致している必要があります（例: `/ask`）。

---

## Step 6: Event Subscriptions を有効化する（Socket Mode では不要）

Socket Mode を使う場合、イベントを受け取るエンドポイント URL の設定は**不要**です。
Slack から slack-router へのコネクションは、slack-router 側から能動的に確立されます。

---

## Step 7: トークンを設定する

トークンは **環境変数で渡すことを推奨**します。`config.yaml` をリポジトリに含めていても、トークンが漏洩するリスクを排除できます。

```bash
cp .env.example .env
```

`.env` を開き、Step 2 と Step 4 で取得したトークンを記入します。

```bash
SLACK_APP_TOKEN=xapp-1-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
SLACK_BOT_TOKEN=xoxb-xxxxxxxxxxxx-xxxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxx
```

> `.env` は `.gitignore` により Git 管理対象外です。絶対にコミットしないでください。

`config.yaml` 側のトークン欄は空のままで構いません。

```bash
cp config.example.yaml config.yaml
# slack.app_token / bot_token は空欄のまま — 環境変数が優先されます
```

---

## Step 8: 起動して接続を確認する

```bash
# 環境変数を読み込んで起動
set -a && source .env && set +a
./slack-router -config config.yaml
```

以下のようなログが出れば接続成功です。

```json
{"level":"INFO","msg":"slack-router starting","routes":2,"max_concurrent_workers":10}
{"level":"INFO","msg":"connecting to slack"}
{"level":"INFO","msg":"connected to slack"}
```

Slack の任意のチャンネルで登録した Slash Command を実行し、ワーカースクリプトが起動することを確認してください。

---

## トラブルシューティング

### `invalid_auth` エラーが出る

- `app_token` / `bot_token` が正しくコピーされているか確認する
- App が対象ワークスペースにインストールされているか確認する

### コマンドを実行しても何も起きない

- Slash Command の **Command** 欄と `config.yaml` の `routes[].command` が一致しているか確認する
- `log_level: "debug"` に変更してログを確認する

### `socket mode is not enabled` エラーが出る

- Slack App の設定画面で **Socket Mode** が有効になっているか確認する
- `app_token` に `xapp-` から始まるトークンを使用しているか確認する（`xoxb-` と混同しないこと）

### ワーカースクリプトが起動しない

- スクリプトに実行権限があるか確認する（`chmod +x ./scripts/ask_llm.sh`）
- スクリプトのパスが `config.yaml` の `routes[].script` と一致しているか確認する
- `log_level: "debug"` にして詳細ログを確認する

---

## トークンの管理について

`config.yaml` にはトークンを平文で記載します。以下の点に注意してください。

- `config.yaml` を Git リポジトリにコミットしない（`.gitignore` に追記する）
- ファイルのパーミッションを制限する（`chmod 600 config.yaml`）
- 必要に応じて環境変数からトークンを読み込む仕組みを検討する
