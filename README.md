# QEMU QAPI Go Client

A type-safe, auto-generated Go client for [QEMU](https://www.qemu.org/)'s [QAPI](https://wiki.qemu.org/Features/QAPI) schema.

This project plugs into **QEMU's official Python-based QAPI generator**, extending it with a **Go backend** to produce idiomatic Go code from QAPI schemas. The result is a native Go client that speaks QEMU’s management protocols (QMP, QGA, etc.) with **zero boilerplate and full schema compliance**.

If you’re building tools for managing or orchestrating QEMU virtual machines — and want type safety and clarity — this is for you.

## 🚀 Features

- **Schema-accurate Go bindings** — full QAPI coverage, directly from `qapi-schema.json`
- **Built on QEMU's own generator** — extended with a Go backend
- **Modular code generation** — per QAPI module, with clear Go packages
- **Asynchronous communication**

## Architecture

This project consists of:
- A **core Go client**
- A **Go backend** for QEMU’s official `qapi-gen.py` generator (written in Python)
- **Generated Go packages** per schema module, with types and methods auto-derived from QAPI

The core handles async messaging:

```text
Client sends:       { "id": 1, "execute": "query-status" }
QEMU replies:       { "id": 1, "return": { "status": "running", ... } }

Events can arrive anytime: { "event": "SHUTDOWN", "timestamp": ... }
```

![Asynchronoous communication](./async-comm.svg)

## Quick start

1. Generate Go bindigs:
```shell
./generate.sh --schema build/qemu/qapi/qapi-schema.json --out-dir generated --package qapi
```

2. Use the generated client in Go:
```go
monitor, monitorErr := client.NewMonitor()
if monitorErr != nil {
    return monitorErr
}

defer monitor.Close()
msgCh := monitor.Start()

monitor.Add("example instance", socketPath)
if req, reqErr := qapi.PrepareQmpCapabilitiesRequest(qapi.QObjQmpCapabilitiesArg{}); reqErr == nil {
    if ch, chErr := monitor.Execute("example instance", client.Request(*req)); chErr == nil {
        res := <-ch
        // ...
    }
}
```

3. Listen to events:
```go
for msg := range msgCh {
    // ...
}
```

[example](./example/) contains an example project that uses client and QAPI generated code to communicate with QEMU QMP.

## Motivation

The primary motivation for this project was the lack of QEMU clients that are fully compliant with the QAPI schema. Existing solutions did not leverage QEMU’s own generator, which is used internally to produce backend code. By extending the official QAPI parser to generate Go clinet code, this project ensures strict schema compliance and seamless integration, enabling Go developers to build reliable tools and automation around QEMU without manual protocol handling.

## Custom Schemas

As long as your schema is compatible with QEMU's generator infrastructure, this will generate Go code for it.
That means it works even with vendor-extended or forked QEMU builds.
