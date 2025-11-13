# Hog scheduler

## Install and deploy on cluster 

#### 1. Install `hog_lockd` daemon and `hog_sched` scheduler

- log into `hog12`
- copy `PKGBUILD` to `/tmp/hogtools` and `cd` there
- run `makepkg -si` (this installs in `/usr/local/bin`)
- run `cd ~dm/ansible_hog` then `sudo ansible-playbook misc-config.yml` to populate to all machines

#### 2. Deploy `hog_lockd` and configure `hog_sched`

- place a `.hogtools` file in the home dir of the executing user (e.g. `hogci`):

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

GOPRIVATE=github.com/groundhogchain/*
GONOSUMDB=github.com/groundhogchain/*
GONOPROXY=github.com/groundhogchain/*

git config --global url."git@github.com:".insteadOf "https://github.com/"
git clone --depth 1 git@github.com:groundhogchain/chuckd.git
cd chuckd

###

go build ./...
go test ./... -covermode=atomic -coverprofile=coverage.out
go tool cover -func=coverage.out
mv coverage.out ../

"""
file_in = []
file_out = ["coverage.out"]

[jobs.env]
GIT_COMMIT = "${{ .git_commit }}"
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
        runs-on: self-hosted
        steps:
            - name: Run hog job
              run: |
		        git clone --depth 1 git@github.com:groundhogchain/chuckd.git
		        hog_sched \
			   	--i="github-${{ github.run_id }}" \
				--key=git_commit --value=${{ github.sha }}
				./scripts/schedules/build_test_cover.toml

            - name: Upload coverage artifact
              uses: actions/upload-artifact@v4
              with:
                  name: coverage
                  path: coverage.out
```
