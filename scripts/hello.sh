#!/usr/bin/env bash
#
# hello.sh — 挨拶サンプルワーカー
#
# stdin から受け取った JSON を元に、コマンドを実行したユーザーへ
# response_url 経由で挨拶メッセージを返す。
#
# 対応コマンド例:
#   /hello              → "こんにちは、@username さん！"
#   /hello おはよう     → "おはよう、@username さん！"
#
# 依存: jq, curl

set -euo pipefail

# --- 依存ツールの存在確認 ---------------------------------------------------
for cmd in jq curl; do
    if ! command -v "$cmd" > /dev/null 2>&1; then
        echo "[hello.sh] ERROR: required command not found: $cmd" >&2
        exit 1
    fi
done

# --- stdin から JSON を読み取る -------------------------------------------
payload=$(cat)

user_id=$(      echo "$payload" | jq -r '.user_id')
text=$(         echo "$payload" | jq -r '.text')
response_url=$( echo "$payload" | jq -r '.response_url')

# --- 必須フィールドの検証 --------------------------------------------------
if [[ -z "$user_id" || "$user_id" == "null" ]]; then
    echo "[hello.sh] ERROR: user_id is missing" >&2
    exit 1
fi
if [[ -z "$response_url" || "$response_url" == "null" ]]; then
    echo "[hello.sh] ERROR: response_url is missing" >&2
    exit 1
fi

# --- response_url の検証（SSRF 対策）--------------------------------------
# https://hooks.slack.com/ から始まるURLのみ許可する
if [[ "$response_url" != "https://hooks.slack.com/"* ]]; then
    echo "[hello.sh] ERROR: response_url is not a valid Slack webhook URL" >&2
    exit 1
fi

# --- メッセージを組み立てる -----------------------------------------------
if [[ -z "$text" || "$text" == "null" ]]; then
    greeting="こんにちは"
else
    greeting="$text"
fi

message="${greeting}、<@${user_id}> さん！ :wave:"

# --- Slack へ返信する -------------------------------------------------------
curl -sSf -X POST "$response_url" \
    -H "Content-Type: application/json" \
    -d "$(jq -n --arg text "$message" '{"response_type": "ephemeral", "text": $text}')"
