# Hog scheduler

## TODO

- [ ] .hogtools config (lockd address, job dir)

## Install and deploy on cluster 

#### 1. Install `hog_lockd` and `hog_sched`

- log into `hog12`
- copy `PKGBUILD` to `/tmp/hogtools` and `cd` there
- run `makepkg -si` (this installs in `/usr/local/bin`)
- run `cd ~dm/ansible_hog` then `sudo ansible-playbook misc-config.yml` to populate to all machines

#### 2. Deploy `hog_lockd` daemon

- place a `.hogtools` file in your home dir

## Usage

Write a TOML spec for a multi-host execution, for instance `build_test_cover.toml` executes one job:

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

[[jobs.env]]
GIT_COMMIT = "${{ github.sha }}"
```

Execute this spec by running:

```sh
hog_sched -i=123xyz build_test_cover.toml
```
Flag `-i` is a required instance id.

### Github CI with `hog_sched`

Make sure the running user has ssh credentials for access to github repos on all machines.

```sh
export HOG_LOCKD="hog12:60006"
export HOG_JOBDIR="/tmp/hogjob"
hog_sched schedule.toml
```
