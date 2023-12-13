# remembr-telegram-bot

## Usage

1. Go to [https://t.me/remembr_bot](https://t.me/remembr_bot)
2. Click "Send message"
3. Send something that you want to be reminded about. For example :

> "Pay the water bill in two days at 9am"

4. The bot will automatically send you a (snoozable) notification saying "Pay the water bill" in two days, at 9am

## Development

1. Run `netlify dev --live`
2. Copy the public URL (should end with `netlify.live`) given by netlify-cli
3. Refresh the webhook: `./refresh-webhook.sh <NETLIFY_LIVE_PUBLIC_URL>`
