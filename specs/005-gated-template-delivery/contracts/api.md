# API Contract: Template Deliveries

## POST `/v1/template-deliveries/download`

JSON body:

```json
{
  "acknowledged": true,
  "configuration": {}
}
```

`configuration` is optional only for the public blank template. Success returns
200 with the DOCX MIME type, binary body, `Content-Disposition: attachment`, and
`Cache-Control: no-store`. Validation errors return 400. Disallowed origins
return 403. Rate limits return 429. Generation failures return 500.

## POST `/v1/template-deliveries/email`

Required header: `Idempotency-Key`.

JSON body:

```json
{
  "acknowledged": true,
  "email": "reader@example.com",
  "configuration": {}
}
```

Success returns 202 only after Resend accepts the message and returns a message
id. Validation errors return 400. Reused keys with a different request return
409 when detected. Disallowed origins return 403. Rate limits return 429.
Generation failures return 500. Unconfigured or failed provider delivery
returns 503.

Neither operation accepts delivery data in path or query parameters.
