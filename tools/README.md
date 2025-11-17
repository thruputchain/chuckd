# Hog scheduler

The "hog scheduler" is a tool for executing multi-host programs on the hog cluster.
A program entails a collection of jobs each executing on a dedicated host in a clean directory.

A program is defined in a single TOML file. It consists of a name and one or more job definitions.
Each job is defined by:
- a unique identifier
- a host to run on
- a shell script to run
- a set of input files or directories, which will be copied from the executing environment to the job directory,
- a set of output files or directories, which will be copied from the job directory back to the executing environment,
- environment variables.

Programs can be executed by any user on any machine. Program execution will not begin until all hosts in the job definitions have been locked exclusvely. Execution terminates with an error as soon as any job does. Otherwise programs terminate successfully. 

Any child subprocesses, except for detached ones, will be terminated whenever a job terminates for any reason.

The job execution environment inherits the ssh-agent of the executing environment. Any ssh keys which are GitHub deploy keys are recognized, and git and ssh are configured to use the appropriate key when cloning.

## Usage

Programs are specified in a TOML file, like `build_test_cover.toml`, for example:

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

To execute this program, the user would run:

```sh
hog_sched -i=123xyz --key=git_commit --value=abcd1234 build_test_cover.toml
```

Here, the flag `-i` is a required unique execution instance id.
Any number of `--key`/`--value` flags can be passed to perform substitutions in the program spec.
In this example, the substitution replaces the string `{{ .git_commit }}` in the spec.

## Github CI and the hog scheduler

Write your CI workflows as scheduler programs and call them from a GitHub workflow as in this example:

```yml
name: Build, test, cover

on:
    push:
        branches: [main]
    pull_request:
        branches: [main]

jobs:
    build-test:
        # pick a runs-on label that corresponds to any available machine.
        # the machine that this workflow runs on is of no consequence,
        # the hog scheduler program will run on the machines defined in its jobs definitions.
        runs-on: self-hosted
        steps:
            # clean up .gitconfig from the previous run of webfactory/ssh-agent
            - name: Clean up .gitconfig
              run: |
                  echo "Cleaning up .gitconfig"
                  rm -f /home/hogci/.gitconfig

            # add all relevant GitHub deploy keys to the workflow
            - uses: webfactory/ssh-agent@v0.9.1
              with:
                  ssh-private-key: |
                      ${{ secrets.CHUCKD_TO_CHUCKD_KEY }}
                      ${{ secrets.CHUCKD_TO_PROTOCOL_GO_KEY }}
                      ${{ secrets.CHUCKD_TO_METRICS_HISTORY_KEY }}

            - uses: actions/checkout@v4

            # execute one or more scheduler programs defined in your repo
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


## How it works

The hog scheduler consists of three binaries:
- `hog_lockd`: a lock server daemon,
- `hog_sched`: the schedule executor command, and
- `hog_fwd`: a utility that forwards git commands to the correct ssh agent key, when GitHub deploy keys are present.

The lock server is deployed on `hog12:60006` and is used by `hog_sched`.

Hog users invoke `hog_sched` from any machine (including non-cluster machines that have ssh access to the cluster) to execute a program.

`hog_sched` expects to find a config file in `~/.hogtools` on the host where a job is executing, containing the location of the lock server and the directory where the job runs, e.g.

```toml
lockd="hog12:60006"
job_dir="/tmp/hogjob-hogci"
```

## Installation

Install the tools on all machines, on `hog12` run:

```sh
./scripts/rebuild_hogtools.sh
```

Populate all machines with a `.hogtools` config file, log as the desired user (on any machine) and run:

```sh
./scripts/populate_hogtools.sh
```

Install `cargo` on all machines:

```sh
for host in "${hosts[@]}"; do ssh $host 'curl -sSf https://sh.rustup.rs | sh -s -- -y' ; done
```
