# Mini Program UI gates

`ui1` is the deterministic loopback-fixture regression gate. `ui1:composed` is a separate local-composed gate that renders the same Mini Program pages in locked Chromium and forwards their HTTP calls to an already-running local `order-api` backed by MySQL.

```sh
ORDER_COMPOSED_API_ORIGIN=http://127.0.0.1:8080 \
MINIPROGRAM_UI_DEPS=/path/to/reused/tools/miniprogram-ui \
npm --prefix tools/miniprogram-ui run ui1:composed
```

The composed runner does not start or mutate MySQL and does not claim an API port. It opens a random loopback CORS proxy, forwards request methods, paths, headers, and bodies unchanged, and reports upstream status codes plus sanitized request IDs. Its HTTP-failure scenario rewrites only the request path to a guaranteed-absent route; the resulting `404` is returned by the real `order-api`, not fabricated by the runner.

The transaction scenario uses the development identity and payment providers only: trusted phone binding, Quote, prepay, server confirm, order result/list/detail, eligible user cancellation, and the local refund worker all go through the real API and MySQL. The runner's `wx.requestPayment` callback is deterministic local UI glue and never contacts WeChat or moves funds. A confirm-path `404` proves that ambiguous HTTP failure keeps the cart and suppresses result navigation.

This is `L3_LOCAL_COMPOSED` evidence only. It is not WeChat DevTools UI2, physical-device UI3, real phone authorization, or real payment evidence.

The healthy local payment provider marks its first queried transaction paid, so it cannot return a deterministic HTTP `202 PENDING`. That state remains covered only by the existing UI0 contract until the backend supplies an explicit non-production NOTPAY mode. Merchant READY/redeem is also outside this runner because it requires lawful pickup-time advancement; this gate does not mutate time or order facts.
