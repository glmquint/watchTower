# Proto contracts and codegen

This folder contains gRPC service contracts and codegen config.

## Files
- `auth.proto` — AuthService contracts: Register, Login, ValidateToken
- `incident.proto` — IncidentService contracts: ListIncidents, CreateIncident
- `buf.yaml`, `buf.gen.yaml` — buf workspace and generation config

## Generate Go code

Make sure you have buf and protoc plugins installed, then run:

```
buf generate
```

Generated code will appear under `proto/gen/` using Go import paths:
- `watchtower/proto/gen/auth/v1` (package `authv1`)
- `watchtower/proto/gen/incident/v1` (package `incidentv1`)

## Importing from services and gateway

In `api/` and `services/*`, add a `replace` to point the `watchtower/proto` module path to the local `../proto` folder, for example:

```
replace watchtower/proto => ../proto
```

Then you can import generated code like:

```
import authv1 "watchtower/proto/gen/auth/v1"
```
