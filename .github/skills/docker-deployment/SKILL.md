---
name: docker-deployment
description: "Use when containerizing apps, creating Dockerfile/docker-compose, wiring infra services (db/api/tunnel), hardening images, and troubleshooting Docker build/network/port conflicts."
argument-hint: "Stack details: language/runtime, services, ports, env vars, tunnel/proxy needs"
---

# Docker Deployment

Build production-grade container stack and fix bring-up failures fast.

## When to Use

- Create Dockerfile for app runtime.
- Create/update docker-compose stack.
- Add infra services (Postgres, Redis, proxy, tunnel).
- Bind host ports (ex: `8000:8000`).
- Resolve Docker build failure (`go mod download`, package fetch, timeout).
- Resolve runtime conflict (port already allocated, container name conflict, unhealthy service).

## Inputs

- Runtime: Go / Python / Node / other.
- Services needed: `api`, `db`, `cache`, `proxy`, `cloudflared`, etc.
- Host ports and internal ports.
- Required env vars and secrets strategy.
- Dev vs prod behavior.

## Core Rules

### MUST

- Use multi-stage builds for production images.
- Run app as non-root user.
- Add health checks for critical services.
- Pin image versions (avoid `latest` for app/db unless user explicitly asks).
- Keep secrets in env vars/secrets manager, never bake into image.
- Use `.dockerignore` to reduce build context.

### NEVER

- Hardcode credentials in Dockerfile/compose.
- Run production app as root.
- Ignore health status before declaring stack ready.
- Keep stale legacy containers that block required ports.

## Procedure

1. Inventory stack

- Detect current Dockerfile, compose files, Makefile/task runner, env files.
- Map required services, ports, and dependencies.

2. Build Dockerfile strategy

- Prod: multi-stage + minimal runtime deps + non-root user.
- Dev: optional separate Dockerfile with reload/debug tools.
- Add retry on dependency download if network is flaky (bounded retries).

3. Compose service design

- Define services with clear responsibilities.
- Add `depends_on` with health conditions when available.
- Map required ports explicitly (`host:container`).
- Add volumes only where needed (db data, logs, source mounts for dev).

4. Wire app config

- Set app env from `.env` / compose env.
- For API + DB, use service DNS in container network (`db:5432`), not `localhost`.
- Keep host access path for local tools (`localhost:<published_port>`).

5. Add operations workflow

- Provide commands/targets:
  - start stack
  - stop stack
  - show status
  - follow logs
  - wait for readiness
- If repository uses Makefile, expose these as `infra-up/down/ps/logs`.

6. Troubleshoot branches

- If build fails with transient network/TLS timeout:
  - retry failed install/download step with bounded backoff.
- If `port already allocated`:
  - detect blocking container/process.
  - remove/stop legacy conflicting container only when it is safe.
- If service unhealthy:
  - inspect logs.
  - run readiness command inside container (`pg_isready`, curl health endpoint).

7. Validate

- `docker compose config` passes.
- stack starts without errors.
- db healthy.
- api reachable on expected host port.
- optional services (tunnel/proxy) running.

## Completion Checks

- Dockerfile follows hardening rules (multi-stage, non-root, no secrets baked).
- compose includes all requested services and port bindings.
- startup command succeeds on clean machine with Docker running.
- troubleshooting notes/commands provided for known failure points.

## Quick Command Checklist

```bash
docker compose -f docker-compose.infra.yml config
docker compose -f docker-compose.infra.yml up -d --build
docker compose -f docker-compose.infra.yml ps
docker compose -f docker-compose.infra.yml logs -f
docker compose -f docker-compose.infra.yml down
```

## Output Contract

When invoked, produce:

1. Updated Dockerfile/compose files.
2. Ops commands (or Makefile targets) to run stack.
3. Explicit readiness checks.
4. Concise fix path for any startup/build failure encountered.
