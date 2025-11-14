# Hog scheduler

## Install and deploy on cluster 

#### 1. Install `hog_lockd` daemon, `hog_sched` scheduler, and `hog_fwd` github ssh forwarder

On `hog12` run:

```sh
./scripts/rebuild_hogtools.sh
```

#### 2. Deploy `hog_lockd` and configure `hog_sched`

- place a `.hogtools` file in the home dir of the executing user (e.g. `hogci`) on all machines:

```toml
lockd="hog12:60006"
job_dir="/tmp/hogjob"
```

## Usage

Write a TOML spec for your experiment, for instance `build_test_cover.toml` executes one job:

```toml
name = "build_test_cover"

[[jobs]]
name = "build_test_cover"
host = "hog6"
script = """

set -x

# configure go

export GOPRIVATE=github.com/groundhogchain/*
export GONOSUMDB=github.com/groundhogchain/*
export GONOPROXY=github.com/groundhogchain/*
export GOPATH=$(pwd)/go
export PATH=$PATH:$GOPATH/bin

echo "working environment"
env

# configure git

git config --global protocol.version 2
# git config --global url."git@github.com:".insteadOf "https://github.com/"

# clone

git clone --depth 1 git@github.com:groundhogchain/chuckd.git
cd chuckd

# build

go build ./...

# test

go test ./... -covermode=atomic -coverprofile=coverage.out

# cover

go tool cover -func=coverage.out

# export result

mv coverage.out ../

"""
file_in = []
file_out = ["coverage.out"]

[jobs.env]
GIT_COMMIT = "{{ .git_commit }}"
```

Execute this spec by running:

```sh
hog_sched -i=123xyz --key=git_commit --value=abcd1234 build_test_cover.toml
```

Flag `-i` is a required instance id.

You can add as many jobs as you like. They must use different hosts. Execution won't start until all hosts are locked (i.e. no other execution can take them).

### Github CI with `hog_sched`

Make sure the running user (i.e. `hogci`) has ssh credentials for access to groundhog github repos on all machines.

Use a github workflow like this example:

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
                  set -x
                  hog_fwd
                  ssh-add -L
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
