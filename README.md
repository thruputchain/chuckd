# chuckd

The coordinator for a Thruput validator, written in Go.

A Thruput validator runs as several cooperating processes. `chuckd` is the
one that ties them together: it disseminates transactions and blocks between
validators, drives consensus through [`scpd`](https://github.com/thruputchain/scp)
(the Stellar Consensus Protocol daemon, run as a child process), and hands
agreed blocks to the execution pipeline over ONC-RPC. The wire format is the
Thruput protocol's XDR, through
[thruput-protocol-go](https://github.com/thruputchain/thruput-protocol-go).

| Directory | Contents |
|-----------|----------|
| `chuckd/` | The `chuckd` command: `sys` (run a system), `bench`, `config`, `deploy` and `hosted-bootstrap` subcommands |
| `system/` | Assemblies of subsystems into runnable systems: the coordinator, the standalone disseminator, and benchmarks |
| `sub/` | Subsystems: `baton` (consensus driver), `scp` (the `scpd` bridge), `storage`, `sched`, `catchup`, `repair`, `cvpool`, `wallet`, `loadgen` |
| `net/` | Networking: `dogwood` (erasure-coded broadcast), `magnolia` (Reed-Solomon coding), `discovery`, `router`, `multiplex` |
| `runtime/` | Process-level concerns: telemetry, pprof, metrics |
| `util/`, `test/` | Shared helpers and test fixtures |
| `scripts/` | Build, packaging and benchmark job scripts |
| `deploy/` | Generators for multi-node benchmark deployments |
| `doc/` | Design notes and operator documentation |

## Building

Requirements: Go (the version in `go.mod`) and, for `scpd`, a Rust toolchain
and the `scp` submodule.

```sh
make config      # git submodules plus go mod download
make build       # ./chuckd.dev and ./scpd.dev
make coordinator # just chuckd
make install     # copy both binaries to $PREFIX/bin as chuckd and scpd
```

`make help` lists every target. `chuckd sys coordinator` finds `scpd` on
`PATH` at startup and runs it as a child process, so install both or neither.

## Testing

```sh
make check   # what CI enforces: go test ./...
make lint    # go vet ./...
make fmt     # go fmt ./...
```

Some integration tests are opt-in through environment variables:
`RUN_CHUCKD_VIRTUAL_TEST=1` runs the multi-node virtual coordinator tests and
`RUN_CHUCKD_RUST_INTEROP_TEST=1` checks signing against the Rust implementation.

## Benchmarks

The cluster benchmarks (`chuckd bench coordinator` on three hosts, the Dogwood
broadcast benchmarks, the virtual coordinator tests) run through the private
[thruputchain/benchmarks](https://github.com/thruputchain/benchmarks)
repository, which holds their hog_sched schedules under `chuckd/`. The per-host
job scripts they run live here under `scripts/schedules/host_jobs/`. Every push
to `main` dispatches the Dogwood benchmarks there through
`.github/workflows/benchmarks.yml`.

## Running

`chuckd --help` documents the global flags (logging, profiling, metrics
address, seed). Each system reads a JSON configuration file whose schema is the
protocol's `Config` XDR type; `chuckd config init` writes a starting point and
`chuckd config group` generates a set of mutually-known validators.

Operator configuration uses unprefixed base58 public keys. Regenerate old hex
validator and quorum configuration with matching ops templates. Secret keys
retain their existing hex format. Hosted bootstrap requires metadata schema 3
and emits hosted runtime-manifest schema 4; regenerate older development
bootstrap state.

## Documentation

- [Dogwood](doc/DOGWOOD.md), the dissemination subsystem, and how to run it
  standalone
- [Component builds](doc/component-builds.md), the manual immutable builds
  used for packaging
- [Troubleshooting](doc/troubleshoot.md)
- [Security notes](doc/security-notes.md), hardening work tracked in the code

## Contributing and security

See [CONTRIBUTING.md](CONTRIBUTING.md). To report a vulnerability, follow
[SECURITY.md](SECURITY.md) rather than opening a public issue.

## License

Apache-2.0. See [LICENSE](LICENSE).
