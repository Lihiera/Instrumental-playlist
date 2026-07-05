# API Reference

## Runtime Settings

The server reads `.env` first, then process environment variables. Process environment variables take precedence.

Required or supported settings:

- `HTTP_ADDR`: HTTP listen address. Default: `:8080`.
- `APPLE_DEVELOPER_TOKEN`: Apple Music Developer Token. This value is never returned by API responses.
- `APPLE_STOREFRONT`: Apple Music storefront code. Default: `us`.
- `INSTRUMENTAL_THRESHOLD`: Instrumental detection threshold from `0` to `1`. Default: `0.75`.

## Endpoints

### `GET /health`

Returns a lightweight health response.

```json
{"status":"ok"}
```

### `GET /v1/config`

Returns public runtime configuration. Secret values are redacted by omission.

```json
{
  "http_addr": ":8080",
  "apple_developer_token_configured": true,
  "apple_storefront": "us",
  "instrumental_threshold": 0.75
}
```

Future Apple Music user-library endpoints will accept a Music User Token through `X-Music-User-Token`.
