# Hukum AI — Backend

AI-powered legal platform. Built with Go (Gin + GORM + PostgreSQL).

> **Cross-reference:** For frontend setup (Next.js, Tailwind, Clerk auth), see [`/frontend/README.md`](../frontend/README.md).

---

## Stripe Subscription Setup

### 1. Create a Stripe account and get API keys

1. Sign up at [stripe.com](https://stripe.com) and go to **Developers → API keys**.
2. Copy your **Secret key** (`sk_live_...` for production, `sk_test_...` for testing).

### 2. Create the Pro plan product and price in Stripe

1. In the Stripe Dashboard, go to **Products → Add product**.
2. Name it **Pro** and add a recurring price of **Rp 299,000 / month** (or your desired amount).
3. Copy the **Price ID** (format: `price_...`).
4. In the database, update the `subscription_plans` row where `slug = 'pro'`:
   ```sql
   UPDATE subscription_plans SET stripe_price_id = 'price_XXXXXXXX' WHERE slug = 'pro';
   ```

### 3. Configure backend environment variables

Add the following to `backend/.env`:

```env
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...   # filled in step 4
FRONTEND_URL=http://localhost:3000
```

### 4. Set up the Stripe webhook

Stripe must be able to reach the backend's webhook endpoint to update subscription state.

#### Option A — Local development (Stripe CLI)

Install the [Stripe CLI](https://stripe.com/docs/stripe-cli) and run:

```bash
stripe listen --forward-to http://localhost:8080/api/subscriptions/webhook
```

The CLI will print a **webhook signing secret** (`whsec_...`). Copy it into `STRIPE_WEBHOOK_SECRET` in `backend/.env`, then restart the backend.

#### Option B — Production / staging (Dashboard)

1. In the Stripe Dashboard, go to **Developers → Webhooks → Add endpoint**.
2. Set the **Endpoint URL** to:
   ```
   https://<your-backend-domain>/api/subscriptions/webhook
   ```
3. Under **Events to send**, select all of the following:
   - `checkout.session.completed`
   - `customer.subscription.created`
   - `customer.subscription.updated`
   - `customer.subscription.deleted`
   - `invoice.paid`
   - `invoice.payment_failed`
4. Click **Add endpoint**.
5. Open the newly created webhook, click **Reveal** next to *Signing secret*, and copy the value (`whsec_...`).
6. Set it as `STRIPE_WEBHOOK_SECRET` in your backend environment (`.env` or the deployment platform's secret manager).

### 5. Webhook event reference

| Event | What the backend does |
|---|---|
| `checkout.session.completed` | Creates an active `Subscription` record for the user |
| `customer.subscription.created` | Updates subscription status |
| `customer.subscription.updated` | Syncs status changes (e.g. trial → active, active → past_due) |
| `customer.subscription.deleted` | Marks the subscription as `canceled` |
| `invoice.paid` | Confirms renewal; updates billing period dates |
| `invoice.payment_failed` | Marks subscription as `past_due` |

### 6. Test the flow end-to-end

Use Stripe's test card `4242 4242 4242 4242` with any future expiry and any CVC to complete a test checkout. After payment, the `checkout.session.completed` webhook fires and the user's subscription is activated in the database.
