#!/usr/bin/env bash
set -euo pipefail

#############################
# USAGE
#############################
usage() {
  cat <<EOF
Usage:
  $0 --host <user@host1> --cmd "<remote_command1>" \
     [--host <user@host2> --cmd "<remote_command2>" ... ]

Each --host must be followed by one --cmd.
Example:
  $0 \
    --host user@h1 --cmd "cd /srv/app && ./run.sh" \
    --host user@h2 --cmd "cd /srv/app && ./run.sh"
EOF
  exit 1
}

#############################
# PARSE ARGUMENTS
#############################
HOSTS=()
CMDS=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --host)
      [[ $# -lt 2 ]] && usage
      HOSTS+=("$2")
      shift 2
      ;;
    --cmd)
      [[ $# -lt 2 ]] && usage
      CMDS+=("$2")
      shift 2
      ;;
    -*|--*)
      echo "Unknown option: $1" >&2
      usage
      ;;
    *)
      echo "Unexpected argument: $1" >&2
      usage
      ;;
  esac
done

if [[ ${#HOSTS[@]} -eq 0 ]]; then
  echo "Error: at least one --host/--cmd pair required." >&2
  usage
fi

if [[ ${#HOSTS[@]} -ne ${#CMDS[@]} ]]; then
  echo "Error: number of --host and --cmd arguments must match." >&2
  exit 1
fi

#############################
# RUN COMMANDS
#############################
PIDS=()

echo "Starting remote processes..."
for i in "${!HOSTS[@]}"; do
  host="${HOSTS[$i]}"
  cmd="${CMDS[$i]}"
  echo "[$i] ${host}: ${cmd}"
  ssh -T "$host" "$cmd" &
  PIDS+=("$!")
done

echo "All remote commands started. Waiting for them to finish..."

exit_code=0
for idx in "${!PIDS[@]}"; do
  pid="${PIDS[$idx]}"
  host="${HOSTS[$idx]}"
  if wait "$pid"; then
    echo "[$idx] $host finished successfully."
  else
    echo "[$idx] $host failed." >&2
    exit_code=1
  fi
done

echo "All remote processes have exited."
exit "$exit_code"
