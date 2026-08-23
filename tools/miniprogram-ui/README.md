# Mini Program UI gates

`ui1` is the deterministic loopback-fixture regression gate. `ui1:composed` is a separate local-composed gate that renders the same Mini Program pages in locked Chromium and forwards their HTTP calls to an already-running local `order-api` backed by MySQL.

```sh
ORDER_COMPOSED_API_ORIGIN=http://127.0.0.1:8080 \
MINIPROGRAM_UI_DEPS=/path/to/reused/tools/miniprogram-ui \
npm --prefix tools/miniprogram-ui run ui1:composed
```

The composed runner does not start or mutate MySQL and does not claim an API port. It opens a random loopback CORS proxy, forwards request methods, paths, headers, and bodies unchanged, and reports upstream status codes plus sanitized request IDs. Its HTTP-failure scenario rewrites only the request path to a guaranteed-absent route; the resulting `404` is returned by the real `order-api`, not fabricated by the runner.

The transaction scenario uses the development identity and payment providers only: trusted phone binding, Quote, prepay, server confirm, order result/list/detail, eligible user cancellation, and the local refund worker all go through the real API and MySQL. The runner's `wx.requestPayment` callback is deterministic local UI glue and never contacts WeChat or moves funds. A confirm-path `404` proves that ambiguous HTTP failure keeps the cart and suppresses result navigation.

To verify durable payment uncertainty separately, start the development API with `ORDER_LOCAL_PAYMENT_AUTO_PAY=false` and run only the explicit pending scenario:

```sh
ORDER_COMPOSED_PAYMENT_EXPECTATION=pending \
ORDER_COMPOSED_API_ORIGIN=http://127.0.0.1:8080 \
MINIPROGRAM_UI_DEPS=/path/to/reused/tools/miniprogram-ui \
npm --prefix tools/miniprogram-ui run ui1:composed
```

Pending mode does not register the four success-mode scenarios. It drives real session, phone binding, Quote, prepay, local `wx.requestPayment`, and two real Confirm requests. Both Confirms must return `202 PENDING`; the page must keep the cart, suppress result navigation, and use a fresh confirm idempotency key for the retry. Omitting `ORDER_COMPOSED_PAYMENT_EXPECTATION` keeps the default four-scenario success gate.

This is `L3_LOCAL_COMPOSED` evidence only. It is not WeChat DevTools UI2, physical-device UI3, real phone authorization, or real payment evidence.

Merchant READY/redeem is outside this runner because it requires lawful pickup-time advancement; this gate does not mutate time or order facts.
