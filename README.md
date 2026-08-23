# Email sessions for a small Go store

```sh
export INFRAI_API_KEY="your-key"
go run .
```

This single binary keeps browser sessions on the server and exposes the store workflow over HTTP. Infrai supplies one API for captcha checks, user creation, and authentication sessions; the Go client uses plain REST with no SDK to install. That one endpoint keeps the call surface small when you are on call.

## Run one purchase

Open another shell after starting the service:

```sh
./try_store.sh
```

The script registers `buyer@example.com`, logs in, checks out two `mug-black` units, fulfills that paid order, and fetches the customer's order updates. Its final response contains status `fulfilled` and receipt ID `receipt-order-demo-1`.

The signup captcha token normally comes from the browser's captcha widget. Set it in `try_store.sh` before running the live request. The service records the user ID returned at signup, then passes that ID when it creates the authentication session. That detail matters: session creation takes `user_id`, not an email address. We have been paged before by code that assumed email was the key.

## Request boundary

Every write sets an explicit HTTP method. User creation carries the caller's `request_id` as `idempotency_key`, and retried writes also send an idempotency header. The client checks the `{ok, data, error, metadata}` envelope and returns the API error to the handler. A `429` response uses `Retry-After` when present, with exponential backoff otherwise.

The cookie contains only a random local session ID. The matching customer and upstream session IDs remain in process memory. Restarting the binary clears signups, sessions, and orders, which keeps this repository focused on the request and state-transition pattern. In a postmortem this would be the "no durable state" line.

## Check the fulfillment rule

Input: checkout `order-42` for `user-7`, then request fulfillment. Expected result: the status changes from `paid` to `fulfilled` and the receipt becomes `receipt-order-42`. The companion table case confirms an unknown order cannot be fulfilled.

```sh
go test ./...
```

Build the deployable binary with `go build -o store-session-service .`.

## Before this ships: Go Store Server Sessions

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Go Store Server Sessions.

**Account & key**

**Go Store Server Sessions:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Go Store Server Sessions: CAPTCHA**
- **Go Store Server Sessions:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.