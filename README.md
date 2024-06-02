# OpenAPI Utils

Several utilities for working with openapi document.

Note that this is still in very early development stage. Current limitations includes but not restricted to:

- Can only bundle reference to `schema`.
- When [bundling](#bundle), the `schema` on non-main files needs to be under `components.schemas` key

## Bundle

Bundle splitted Open API files into one file while trying to persist all their `$ref` use.

```bash
go run -mod=mod github.com/telkomindonesia/openapi-utils/cmd/bundle <path-to-main-spec> [<path-to-new-spec>]
```

For testing the functionality, you can use [spec inside testdata directory](./cmd/bundle/testdata/profile/).

## Proxy

Create a new spec by picking operations from other specs. The main purpose was to derive an OpenAPI spec for an [api-gateways or backend-for-frontends](https://microservices.io/patterns/apigateway.html) using OpenAPI spec of services behind it. It introduces a new `x-proxy` extension.

```bash
go run -mod=mod github.com/telkomindonesia/openapi-utils/cmd/bundle <path-to-proxy-spec> [<path-to-new-spec>]
```

For testing the functionality, you can use [specs inside testdata directory](./cmd/proxy/internal/proxy/testdata/spec-proxy.yml).

Next iteration will also include the ability to generate go code that utilize `httputil.ReverseProxy`, inspired by [oapi-codegen](https://github.com/deepmap/oapi-codegen).
