# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-03-14

### Added

- **Slack Socket Mode 接続** — インバウンドポート開放不要でイベントを受信
- **YAML ルーティングテーブル** — コマンドとスクリプトの紐づけを設定ファイルで管理
- **安全なパラメータ渡し** — コマンドメタデータを `stdin` 経由の JSON で渡す（argv を使わずプロセス一覧からの情報漏洩を防止）
- **グローバル / ルート別同時実行制御** — チャンネルセマフォ方式で DoS を防止
- **タイムアウト強制停止** — SIGTERM → 5秒待機 → SIGKILL でプロセスツリーごと終了
- **ACL（アクセス制御リスト）** — ルート単位で allow/deny チャンネル・ユーザーを設定可能
- **設定可能なユーザー向けメッセージ** — 拒否・輻輳時のメッセージを config.yaml から変更可能
- **ephemeral 通知** — 拒否・エラーメッセージはリクエストしたユーザーにのみ表示
- **環境変数によるトークン注入** — `SLACK_APP_TOKEN` / `SLACK_BOT_TOKEN` で config.yaml をトークンフリーに保てる
- **起動時スクリプト検証** — 存在確認・実行権限・world-writable チェックをデーモン起動時に実施
- **スクリプトパスの絶対化** — config ファイル基準で解決し CWD 非依存に
- **構造化 JSON ログ** — `log/slog` による version / commit / PID などのフィールド付きログ
- **グレースフルシャットダウン** — SIGINT/SIGTERM 受信後、実行中ワーカーの完了を待機してから終了
- **クロスプラットフォームビルド** — macOS (amd64/arm64)・Linux (amd64/arm64)・Windows (amd64) に対応
- **ビルド時バージョン埋め込み** — `git describe --tags` の結果を `-ldflags` でバイナリに埋め込み
- **サンプルスクリプト** — `scripts/hello.sh`（挨拶スクリプト）を同梱

[Unreleased]: https://github.com/magifd2/slack-router/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/magifd2/slack-router/releases/tag/v0.1.0
