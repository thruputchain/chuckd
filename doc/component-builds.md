# Manual component builds

This workflow publishes coordinator artifacts for packaging without a production
component release. It runs only from `main` and accepts only ancestors of `main`.
It does not publish crates or deploy a bundle.

## Publish one build

Use a full source commit for reproducible dispatch:

```bash
gh workflow run component-build.yml --repo thruputchain/chuckd --ref main \
  -f target_commit=<40-character-main-ancestor>
```

The component version is `0.0.0-main.g<first-12-source-characters>`. The workflow
uses this version in the binary and descriptor. Operators cannot override it.
Tagged production releases retain their own component version.

The prerelease coordinate is `packaging-main-<40-character-source-commit>`.
It contains two coordinator archives and `groundhog-component-build.json`.
The descriptor records the full source commit, version, architecture, asset
roles, and SHA-256 digests. Pin the descriptor's exact byte digest in packaging.
This first slice uses authenticated GitHub release assets and checksum pinning.
Signed build attestations and consumer attestation verification remain deferred.
A successful publication is not qualification or promotion approval.

## Reruns and partial publication

Manual publications are serialized. An existing coordinate is refused, not
rebuilt or overwritten. This first slice deliberately does not implement
verification-only successful reruns or automatic partial-publication recovery.
A retry cannot be used to repair an immutable coordinate.

If publication fails after creating a draft or tag, stop for operator review.
Do not delete or overwrite the existing coordinate. Keep the run's artifacts
and error evidence. Source changes require a new source commit. A toolchain-only
rebuild needs a separately approved build-input identity scheme.

## Measurements

Both architectures upload a measurement JSON even when a build fails. It records
build duration, CPU time, visible CPU-hour estimate, artifact size, and retry
status. Visible OS CPUs need not equal the Namespace runner allocation, so
`allocatedCpuHours` remains unknown. `queueDelaySeconds` measures workflow creation to build-runner start,
including prerequisite jobs, not isolated runner queue time. Cache hit rate is
unknown because the cache action does not expose lookup and hit counts.

These measurements inform a later cadence decision. This workflow has no
schedule or per-merge trigger.

## Local checks

```bash
python3 -m unittest discover -s .github/tests -p 'test_*.py'
```

The tests use temporary directories and mocked GitHub/Git commands. They do not
publish, build component source, or require credentials. Linux CI also runs these
tests before the native coordinator builds.
