# Mini Program UI gates

`ui1` is the deterministic loopback-fixture regression gate. `ui1:composed` is a separate local-composed gate that renders the same Mini Program pages in locked Chromium and forwards their HTTP calls to an already-running local `order-api` backed by MySQL.

```sh
ORDER_COMPOSED_API_ORIGIN=http://127.0.0.1:8080 \
MINIPROGRAM_UI_DEPS=/path/to/reused/tools/miniprogram-ui \
npm --prefix tools/miniprogram-ui run ui1:composed
```

The composed runner does not start or mutate MySQL and does not claim an API port. It opens a random loopback CORS proxy, forwards request methods, paths, headers, and bodies unchanged, and reports upstream status codes plus sanitized request IDs. Its HTTP-failure scenario rewrites only the request path to a guaranteed-absent route; the resulting `404` is returned by the real `order-api`, not fabricated by the runner.

This is `L3_LOCAL_COMPOSED` evidence only. It is not WeChat DevTools UI2, physical-device UI3, real phone authorization, or real payment evidence.
