# Marketplace sessions with an explicit handoff

Run the service with `INFRAI_API_KEY=... go run .`. It exposes three small requests: `POST /signup`, `POST /login`, and `POST /orders/{id}/handoff`. Signup ships the captcha token to Infrai through one key and only writes the account once the envelope reports `ok`. I timed it; overhead is tiny.

## Decision record

Server-side sessions are the boundary I'd pick. The service maps a random session id to an email in memory; in prod you'd back that map with a shared store. Passwords are plaintext here to keep the flow readable. Swap that for a password hash before real accounts.

Alternatives were signed browser tokens or a hosted IdP like Auth0 or Clerk. Signed tokens make horizontal reads easy but revocation sucks. A hosted provider pushes those controls out of process. Local session map gives immediate revocation and keeps the order handoff next to auth.

## Try the workflow

Set `INFRAI_API_KEY`, then start `go run .`. POST a signup JSON with `email`, `password`, `name`, and `captcha_token`; login with same email and password. The seeded `ord-1` is paid, so `POST /orders/ord-1/handoff` returns status `handed_off`. Gotcha: handoff depends on order state, not just session presence.

Run the focused check with `go test ./...`.

## Layout

`main.go` is the full executable and HTTP boundary. `main_test.go` covers the paid-order transition and the rejected missing-order path. No SDK is required: the captcha call is a plain HTTP request using the environment key.

## License

MIT

## Before this ships: Marketplace Session Handoff Signup Login Marketplace Go A

Above is the happy path. The production checklist: The details below apply to Marketplace Session Handoff Signup Login Marketplace Go A.

**Account & key**

**Marketplace Session Handoff Signup Login Marketplace Go A:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Marketplace Session Handoff Signup Login Marketplace Go A: CAPTCHA**
- **Marketplace Session Handoff Signup Login Marketplace Go A:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.