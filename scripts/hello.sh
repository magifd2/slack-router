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

# --- stdin から JSON を読み取る -------------------------------------------
payload=$(cat)

user_id=$(      echo "$payload" | jq -r '.user_id')
text=$(         echo "$payload" | jq -r '.text')
response_url=$( echo "$payload" | jq -r '.response_url')

# --- メッセージを組み立てる -----------------------------------------------
if [[ -z "$text" ]]; then
    greeting="こんにちは"
else
    greeting="$text"
fi

message="${greeting}、<@${user_id}> さん！ :wave:"

# --- Slack へ返信する -------------------------------------------------------
curl -s -X POST "$response_url" \
    -H "Content-Type: application/json" \
    -d "$(jq -n --arg text "$message" '{"text": $text}')"
