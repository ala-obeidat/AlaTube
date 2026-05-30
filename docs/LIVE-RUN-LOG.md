# Live Run Log

| Time (UTC) | Method | URL | Status | Latency | Request ID | Notes |
|---|---:|---|---:|---:|---|---|
| Note | | | | | | Two local curl harness attempts failed before sending a usable request and were removed from this live-request log. |
| 2026-05-30T10:52:37.4875066Z | GET | https://alatube-api.alaobeidat.com/api/health | 200 | 0.206497s | codex-948d2c9f-1bf4-4319-8472-638d52718b39 | baseline health |
| 2026-05-30T10:53:03.3699336Z | GET | https://alatube-api.alaobeidat.com/api/health | 200 | 0.235526s | codex-e1f2ec8c-347e-4579-ae9f-de783dd132de | health after harness fix |
| 2026-05-30T10:53:03.6721562Z | OPTIONS | https://alatube-api.alaobeidat.com/api/analyze | 204 | 0.243045s | codex-8455dbb3-fe6a-4da4-9de5-c24601ad756d | allowed origin preflight |
| 2026-05-30T10:53:04.2957221Z | OPTIONS | https://alatube-api.alaobeidat.com/api/analyze | 204 | 0.591911s | codex-309d56e5-1dc4-4dea-90c1-d234ebcac056 | foreign origin preflight |
| 2026-05-30T10:53:04.5483013Z | POST | https://alatube-api.alaobeidat.com/api/analyze | 400 | 0.202619s | codex-2e10f7a6-2ec9-4378-9d88-a95760eb1d02 | reject playlist only |
| 2026-05-30T10:53:04.8649233Z | POST | https://alatube-api.alaobeidat.com/api/analyze | 400 | 0.242748s | codex-ecc14ed1-15ee-4727-9aec-6dcc59504193 | malformed json |
| 2026-05-30T10:53:05.2481674Z | POST | https://alatube-api.alaobeidat.com/api/analyze | 400 | 0.313678s | codex-5c1e313b-1848-43f2-b594-2d53c0d216f1 | unknown json field |
| 2026-05-30T10:53:05.5481316Z | POST | https://alatube-api.alaobeidat.com/api/analyze | 415 | 0.233802s | codex-d665f960-5c24-44f1-bb81-5d97e1d7fd26 | wrong content type |
| 2026-05-30T10:53:38.0186346Z | POST | https://alatube-api.alaobeidat.com/api/analyze | 400 | 0.205906s | codex-077076e4-2b9a-43f9-84af-abe60ee194ea | reject playlist only rerun still escaped intentionally? |
| 2026-05-30T10:53:38.3429882Z | POST | https://alatube-api.alaobeidat.com/api/analyze | 400 | 0.244131s | codex-bbfb7e27-324b-4e67-b030-8d19e694eeb7 | reject playlist only valid json |
| 2026-05-30T10:53:38.6229758Z | POST | https://alatube-api.alaobeidat.com/api/analyze | 400 | 0.219853s | codex-937dd2f8-f8d6-4ecb-be0d-b9952d4ee9f5 | unknown json field valid json |
| 2026-05-30T10:53:38.8845902Z | GET | https://alatube-api.alaobeidat.com/api/health | header-capture | n/a | codex-55fd8ec0-292a-472c-9ce9-5427ac71ebb9 | full headers in docs/headers-health.txt |
| 2026-05-30T10:53:39.1595568Z | OPTIONS | https://alatube-api.alaobeidat.com/api/analyze | header-capture | n/a | codex-710e73e4-b6b2-4155-b010-25bc094e4f8b | allowed CORS headers in docs/headers-cors-allowed.txt |
| 2026-05-30T10:53:39.4510612Z | OPTIONS | https://alatube-api.alaobeidat.com/api/analyze | header-capture | n/a | codex-7c6504ef-ebec-4961-b413-acffc165103e | foreign CORS headers in docs/headers-cors-evil.txt |
| 2026-05-30T10:54:28.9448809Z | POST | https://alatube-api.alaobeidat.com/api/analyze | 400 | 0.208767s | codex-7256ad97-ef3f-426f-b566-1bb121d20388 | reject playlist only valid json |
| 2026-05-30T10:54:29.3281577Z | POST | https://alatube-api.alaobeidat.com/api/analyze | 400 | 0.255505s | codex-fe4b3f1d-e680-469b-874a-23553477b77c | unknown json field valid json |
| 2026-05-30T10:54:29.6185407Z | POST | https://alatube-api.alaobeidat.com/api/analyze | 400 | 0.224086s | codex-a6dc7c2d-c5a9-4b4f-b582-31ed1664a58e | allowlist bypass userinfo host |
| 2026-05-30T10:54:34.9565427Z | POST | https://alatube-api.alaobeidat.com/api/analyze | 200 | 5.245425s | codex-67a0a424-e58d-4ee0-b5bd-fbe91900d274 | allowlist bypass port host |
| 2026-05-30T10:54:40.2749969Z | POST | https://alatube-api.alaobeidat.com/api/analyze | 200 | 5.254194s | codex-2c9fd3ea-d7df-4837-a263-688ea0834746 | happy analyze test video |
| 2026-05-30T10:55:07.8884766Z | POST | https://alatube-api.alaobeidat.com/api/jobs | 202 | 0.223336s | codex-aa5b3d9a-263e-4e9a-b57f-b7580517dc87 | argument-injection-looking format one-shot |
| 2026-05-30T10:55:08.1543858Z | DELETE | https://alatube-api.alaobeidat.com/api/jobs/652f3c09c2668ac778388b0e31f46ec5 | 204 | 0.198665s | codex-dc0e93d4-9872-4808-abbd-f94037ba0e79 | cleanup injection probe job |
| 2026-05-30T10:55:08.4624130Z | GET | https://alatube-api.alaobeidat.com/api/health | 200 | 0.239310s | codex-a529a007-1218-48e4-9fd1-9336f08ed51e | health after injection-looking job probe |
| 2026-05-30T10:55:30.9753188Z | POST | https://alatube-api.alaobeidat.com/api/jobs | 202 | 0.214862s | codex-5dadc484-10f4-4345-ba03-8e9a9f0056d1 | argument-injection config-location one-shot |
| 2026-05-30T10:55:31.2820338Z | DELETE | https://alatube-api.alaobeidat.com/api/jobs/4a8ff945e68b23e07055b87965f7642c | 204 | 0.235662s | codex-6e4160be-4d9b-44e6-ba54-24ddb92e39f8 | cleanup argument-injection config-location one-shot |
| 2026-05-30T10:55:31.5912112Z | GET | https://alatube-api.alaobeidat.com/api/health | 200 | 0.237291s | codex-30f7b176-10c8-48ba-9246-5468c55ec0ec | health after argument-injection config-location one-shot |
| 2026-05-30T10:55:32.1751514Z | POST | https://alatube-api.alaobeidat.com/api/jobs | 202 | 0.189851s | codex-88a4878a-c784-4251-bd29-5a3b866c698f | argument-injection load-info-json one-shot |
| 2026-05-30T10:55:32.4420136Z | DELETE | https://alatube-api.alaobeidat.com/api/jobs/d60ecd716af5ab477c7523131361e077 | 204 | 0.235650s | codex-5504571d-9490-4a19-975b-de47f3178dfd | cleanup argument-injection load-info-json one-shot |
| 2026-05-30T10:55:32.7400251Z | GET | https://alatube-api.alaobeidat.com/api/health | 200 | 0.250416s | codex-01580c69-13ed-4ea9-ae6b-6f7e338a2507 | health after argument-injection load-info-json one-shot |
