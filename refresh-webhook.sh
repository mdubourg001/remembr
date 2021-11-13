# making sure bot token is known
source ./.env
bot_token=$(echo $REMEMBR_TELEGRAM_BOT_TOKEN)

# deleting previous webhook
curl --request POST --url https://api.telegram.org/bot$bot_token/deleteWebhook --header 'content-type: application/json' --data "{\"drop_pending_updates\": true}"

# adding new webhook
webhook_url="$1/.netlify/functions/remembr"
curl --request POST --url https://api.telegram.org/bot$bot_token/setWebhook --header 'content-type: application/json' --data "{\"url\": \"$webhook_url\"}"