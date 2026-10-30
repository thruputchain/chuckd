# Contributing

Thanks for your interest in chuckd. Bug reports, questions and pull requests
are welcome.

## Before you start

- Check the open issues to avoid duplicate work.
- For anything beyond a small fix, open an issue first so the change can be
  discussed before you spend time on it.
- Security problems go through [SECURITY.md](SECURITY.md), not the issue
  tracker.

## Development

```sh
make config   # submodules and go mod download
make build    # chuckd.dev and scpd.dev
make check    # go test ./...
make lint     # go vet ./...
make fmt      # gofmt
```

See the [README](README.md) for what the pieces are and how they fit together.

## Pull requests

- Branch from `main` and keep each pull request to one change.
- Run `make fmt`, `make lint` and `make check` before pushing. CI runs the
  test suite on every pull request.
- Add or update tests for behavior you change.
- Write a commit message that explains why the change is needed, not only
  what it does.
- Keep generated code (for example `system/echo_bench/generated.go`) as the
  generator emits it rather than editing it by hand.

## License

By contributing you agree that your contributions are licensed under the
[Apache License 2.0](LICENSE) that covers this project.
