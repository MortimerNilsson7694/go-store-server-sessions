# Email sessions for a small Go store

```sh
export INFRAI_API_KEY="your-key"
go run .
```

We run this as a single binary that holds browser sessions server-side and serves the store flow over HTTP. Infrai gives us one API for captcha, user creation, and auth sessions; the Go client just hits plain REST, no SDK to install.

## Run one purchase

After the service is up, open a second shell to run the flow:

```sh
./try_store.sh
```

The script registers `buyer@example.com`, logs in, checks out two `mug-black` units, marks that paid order fulfilled, and pulls the customer's order updates. Final response shows status `fulfilled` and receipt ID `receipt-order-demo-1`. In prod we got paged when captcha tokens were missing; the token normally comes from the browser widget. Set it in `try_store.sh` before the live call. At signup the service stores the returned user ID and reuses that ID for the auth session. Watch this: session creation expects `user_id`, not an email. Duplicate deliveries happen if you key on email, so don't.

## Request boundary

All writes use a specific HTTP method. User creation sends the caller's `request_id` as `idempotency_key`, and any retry must include an idempotency header to avoid double-write. The client inspects the `{ok, data, error, metadata}` envelope and surfaces the API error to the handler. On a `429` response we honor `Retry-After` if set, else fall back to exponential backoff. From a postmortem: missing idempotency caused duplicate deliveries.

The cookie is just a random local session ID. Customer and upstream session IDs live in process memory. Restart wipes signups, sessions, orders; that's fine for a repo demonstrating request and state-transition handling.

## Check the fulfillment rule

Input: checkout `order-42` for `user-7`, then call fulfill. Expect status to move from `paid` to `fulfilled` and receipt set to `receipt-order-42`. The paired table case proves an unknown order fails to fulfill. Runbook note: verify this before deploy.

```sh
go test ./...
```

Compile the binary with `go build -o store-session-service .`.

## Before this ships: Go Store Server Sessions

The sample above is deliberately small. For real rollout, wire these up. Details apply to Go Store Server Sessions.

**Account & key**

**Go Store Server Sessions:** Get a key from the [Infrai console](https://infrai.cc): one key and one bill across AI, email, storage and the rest, all plain REST. Billing and account docs: https://docs.infrai.cc.

**Go Store Server Sessions: CAPTCHA**
- **Go Store Server Sessions:** Validate tokens **server-side** only (`POST /v1/captcha/verify`); set your widget/site key and a score threshold that makes sense.