# making sure bot token is known
source ./.env
telegram_host="https://api.telegram.org/bot$REMEMBR_TELEGRAM_BOT_TOKEN"

echo $telegram_host

# deleting previous webhook
curl --request POST --url $telegram_host/deleteWebhook --header 'content-type: application/json' --data "{\"drop_pending_updates\": true}"

# adding new webhook
webhook_url="$1/.netlify/functions/remembr"
curl --request POST --url $telegram_host/setWebhook --header 'content-type: application/json' --data "{\"url\": \"$webhook_url\"}"