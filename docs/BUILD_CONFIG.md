## Build configuration (optional smaller binary)

You can control **which channels are compiled into the binary** to reduce size. This is done **before** the build by parsing a config file and generating which channel packages get compiled in.

### How it works

1. **Config**: Create `build.yaml` in the repo root (copy from `build.yaml.example`).
2. **Allowlist**: Set `channels.include` to the channel names you want to compile in (e.g. `[telegram, whatsapp]`).
3. **Generate**: Run the generator so it writes `cmd/picoclaw/internal/gateway/channels_imports.go` with only the selected channel imports.
4. **Build**: Run `make build`. The generator is run automatically as part of `make build`.

If there is no `build.yaml`, or `channels.include` is empty / omitted, **all discovered channels** under `pkg/channels/` are included by default.

### Quick start

```bash
# Copy example and edit (list channels to include)
cp build.yaml.example build.yaml

# Edit build.yaml, e.g.:
#   channels:
#     include: [telegram, whatsapp]

# Build (generate runs first and overwrites channel imports)
make build
```

Or without Make:

```bash
go run ./scripts/genbuild   # generate channel imports from build.yaml
go build -o picoclaw ./cmd/picoclaw
```

### Channel names

Channel names in `channels.include` must match the directory name under `pkg/channels/`. The generator **automatically discovers** all channel packages by reading the `pkg/channels/` directory, so you usually don't need to touch any code when adding a new channel—just create `pkg/channels/<name>/` and, if desired, list `<name>` in `channels.include`.



Optional denylist options also work:

- `channels.skip: [telegram, whatsapp]`

If `channels.include` is set, it **wins** over any skip settings.
