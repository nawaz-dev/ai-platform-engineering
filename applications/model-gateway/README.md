# Multi-model Go gateway

This gateway routes OpenAI-compatible `/v1/chat/completions` requests based on the client `model` field and exposes a gateway-level `/v1/models` endpoint. It supports upstream SSE pass-through.


## To run the application locally
```
go fmt ./...
go mod tidy
go test ./...
go build ./...
go run ./cmd/gateway
```

or 

```
make run
```
## Quickstart

1. `cp .env.example .env` and set a secure `GATEWAY_API_KEY`.
2. Edit `configs/models.json` to include only available backends; delete the example NIM entry unless you have NIM and `NIM_API_KEY` configured.
3. `docker compose up -d --build`
4. `curl -H "Authorization: Bearer YOUR_KEY" http://localhost:3000/v1/models`
5. `curl -H "Authorization: Bearer YOUR_KEY" -H 'Content-Type: application/json' -d '{"model":"qwen-small","messages":[{"role":"user","content":"Hello"}]}' http://localhost:3000/v1/chat/completions`

Model aliases and upstream model IDs are independent. Mount `configs/models.json` into the container; after changing it, run `docker compose restart model-gateway`. No image rebuild required.

### Limitations

- OpenAI-compatible chat API only (plus `/v1/models`); no API translation for native Ollama endpoints.
- No load balancing, failover, Redis rate limiting, dynamic reload, or tracing yet.
- API keys are a single static gateway secret. Do not expose to public internet without TLS and stronger operational controls.
- Streaming passes through upstream SSE; upstream `model` IDs in responses are not rewritten to aliases.
- The container must be able to reach each configured upstream network address.


```

Merge this with your existing Compose configuration, retaining the model-server networking and volume mount for `configs/models.json`. Redis is not exposed publicly.

## 5. Test rate limiting

Temporarily set:

```dotenv
RATE_LIMIT_RPM=5
```

Rebuild and restart:

```bash
go mod tidy
go test ./...
docker compose up -d --build
```

Send six authenticated requests within one minute:

```bash
for i in {1..6}; do
  curl -s -o /dev/null -w "Request $i: HTTP %{http_code}\n" \
    -H "Authorization: Bearer YOUR_GATEWAY_API_KEY" \
    http://localhost:3000/v1/models
done
```

Expected results, assuming no other requests have consumed the quota:

```text
Request 1: HTTP 200
Request 2: HTTP 200
Request 3: HTTP 200
Request 4: HTTP 200
Request 5: HTTP 200
Request 6: HTTP 429
```

This test counts `/v1/models` requests as well as inference requests. The quota resets at the next UTC minute boundary.

## Next milestone

Once verified, our progress will be:

| Component | Status |
|---|---|
| Go Model Gateway | Completed |
| Multi-model routing | Completed |
| API-key authentication | Completed |
| Redis rate limiting | Ready to implement |
| Per-client API keys and quotas | Next |
| Prometheus metrics | Pending |
| Kubernetes/Helm/Argo CD | Pending |

After this, I recommend **multiple client API keys with independent quotas**, followed by Prometheus observability.

That will give you a reusable gateway capable of managing different applications, model servers, and usage limits.