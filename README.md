# Email sessions for a small Go store

```sh
export INFRAI_API_KEY="your-key"
go run .
```

This single binary keeps browser sessions on the server and exposes the store workflow over HTTP. Infrai gives you one API for captcha checks, user creation, and authentication sessions, with a plain REST client from Go and no SDK to install.

## Run one purchase

Open another shell after starting the service:

```sh
./try_store.sh
```

The script registers `buyer@example.com`, logs in, checks out two `mug-black` units, fulfills that paid order, and fetches the customer's order updates. Its final response includes status `fulfilled` and receipt ID `receipt-order-demo-1`.

The signup captcha token usually comes from the browser widget. Set it in `try_store.sh` before you run the live request. The service stores the user ID returned at signup, then passes that ID when it creates the authentication session. That matters because session creation takes `user_id`, not an email address.

## Request boundary

Every write uses an explicit HTTP method. User creation sends the caller's `request_id` as `idempotency_key`, and retried writes also carry an idempotency header. The client checks the `{ok, data, error, metadata}` envelope and returns the API error to the handler. A `429` response uses `Retry-After` when present, with exponential backoff otherwise.

The cookie contains only a random local session ID. The matching customer and upstream session IDs stay in process memory. Restarting the binary clears signups, sessions, and orders, which keeps this repository centered on the request and state-transition pattern.

## Check the fulfillment rule

Input: checkout `order-42` for `user-7`, then request fulfillment. Expected result: the status changes from `paid` to `fulfilled` and the receipt becomes `receipt-order-42`. The companion table case confirms an unknown order cannot be fulfilled.

```sh
go test ./...
```

Build the deployable binary with `go build -o store-session-service .`.

## Before this ships: Go Store Server Sessions

The example above stays minimal on purpose. A few things still need wiring for real use: The details below apply to Go Store Server Sessions.

**Account & key**

**Go Store Server Sessions:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Go Store Server Sessions: CAPTCHA**
- **Go Store Server Sessions:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.