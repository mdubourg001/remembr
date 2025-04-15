# making sure bot token is known
source ./.env
telegram_host="https://api.telegram.org/bot$REMEMBR_TELEGRAM_BOT_TOKEN"

echo "Telegram bot host is $telegram_host\n"

# deleting previous webhook
echo "Deleting previous webhook...\n"
# curl --request POST --url $telegram_host/deleteWebhook --header 'content-type: application/json' --data "{\"drop_pending_updates\": true}"
curl --request POST --url $telegram_host/setwebhook --header 'content-type: application/json' --data "{\"url\": \"\"}"

# adding new webhook...
echo "Setting new webhook...\n" 
webhook_url="$1/.netlify/functions/remembr"
curl --request POST --url $telegram_host/setWebhook --header 'content-type: application/json' --data "{\"url\": \"$webhook_url\"}"