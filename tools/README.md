# Hog scheduler

A tool, called `hog_sched`, executes "schedules" on behalf of the user.
A schedule specifies multiple jobs, given as shell scripts, to be executed
each on a dedicated hog machine.

## How it works

The hog scheduler consists of three binaries:
- `hog_lockd`: a lock server daemon,
- `hog_sched`: the schedule executor, and
- `hog_fwd`: a utility that recognizes github deploy keys among the current ssh agent keys and configures `git` and `ssh` to use them with their repective repos.

The lock server is deployed on `hog12:60006` and is used by `hog_sched`.

Hog users (such as `hogci`) invoke `hog_sched` (from any machine) to execute a schedule.

`hog_sched` expects to find a config file in `~/.hogtools`, containing the location of the lock server and the directory where jobs run:

```toml
lockd="hog12:60006"
job_dir="/tmp/hogjob-hogci"
```

## Installation

To install the tools on all machines, on `hog12` run:

```sh
./scripts/rebuild_hogtools.sh
```

To populate all machines with a `.hogtools` config file, log as the desired user (on any machine) and run:

```sh
./scripts/populate_hogtools.sh
```

## Usage

Write a TOML spec for your schedule, for instance `build_test_cover.toml` executes one job:

```toml
name = "build_test_cover"

[[jobs]]
name = "build_test_cover"
host = "hog6"
script = """

set -x
git config --global protocol.version 2
git clone --depth 1 git@github.com:groundhogchain/chuckd.git
cd chuckd

source ./scripts/setup_go.sh

go build ./...
go test ./... -covermode=atomic -coverprofile=coverage.out
go tool cover -func=coverage.out

mv coverage.out ../

"""
file_in = []
file_out = ["coverage.out"]

[jobs.env]
GIT_COMMIT = "{{ .git_commit }}"
```

Execute this schedule by running:

```sh
hog_sched -i=123xyz --key=git_commit --value=abcd1234 build_test_cover.toml
```

Flag `-i` is a required unique instance id.

You can add as many jobs as you like to a schedule. They must use different hosts. Execution won't start until all hosts are locked (i.e. no other schedule execution can use them).

## Github CI with `hog_sched`

Use a github workflow similar to this example:

```yml
name: Build, test, cover

on:
    push:
        branches: [main]
    pull_request:
        branches: [main]

jobs:
    build-test:
        runs-on: hog7
        steps:
            - name: Clean up .gitconfig
              run: |
                  echo "Cleaning up .gitconfig"
                  rm -f /home/hogci/.gitconfig

            - uses: webfactory/ssh-agent@v0.9.1
              with:
                  ssh-private-key: |
                      ${{ secrets.CHUCKD_TO_CHUCKD_KEY }}
                      ${{ secrets.CHUCKD_TO_PROTOCOL_GO_KEY }}
                      ${{ secrets.CHUCKD_TO_METRICS_HISTORY_KEY }}

            - uses: actions/checkout@v4

            - name: Run hog schedule
              run: |
                  rm -rf chuckd
                  git -c protocol.version=2 clone --depth 1 git@github.com:groundhogchain/chuckd.git
                  hog_sched \
                    --i="github-${{ github.run_id }}" \
                    --key=git_commit --value=${{ github.sha }} \
                    ./scripts/schedules/build_test_cover.toml

            - name: Upload coverage artifact
              uses: actions/upload-artifact@v4
              with:
                  name: coverage
                  path: coverage.out
```
