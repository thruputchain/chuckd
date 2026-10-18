export GOPRIVATE=github.com/thruputchain/*
export GONOSUMDB=github.com/thruputchain/*
export GONOPROXY=github.com/thruputchain/*
export GOPATH=$(pwd)/__go__
export PATH=$PATH:$GOPATH/bin

# Repositories transferred to thruputchain don't match the groundhogchain
# deploy keys forwarded to the hosts, so fetch them over HTTPS with the
# read-only PAT. GIT_CONFIG_* applies only to this shell and its children,
# leaving the host's git config alone.
if [ -n "${READ_ACCESS_PAT:-}" ]; then
  _thruput_base="https://x-access-token:${READ_ACCESS_PAT}@github.com/thruputchain/"
  export GIT_CONFIG_COUNT=3
  export GIT_CONFIG_KEY_0="url.${_thruput_base}.insteadOf" GIT_CONFIG_VALUE_0="https://github.com/thruputchain/"
  export GIT_CONFIG_KEY_1="url.${_thruput_base}.insteadOf" GIT_CONFIG_VALUE_1="git@github.com:thruputchain/"
  export GIT_CONFIG_KEY_2="url.${_thruput_base}.insteadOf" GIT_CONFIG_VALUE_2="ssh://git@github.com/thruputchain/"
  unset _thruput_base
fi
