parse_flag_j() {
  job=""
  for arg in "$@"; do
    case $arg in
      -j=*|--j=*)
        job="${arg#*=}"
        shift
        ;;
    esac
  done

  if [[ -z "${job:-}" ]]; then
    echo "Error: missing required flag -j=STRING or --j=STRING" >&2
    echo "Usage: $0 -j=STRING" >&2
    exit 1
  fi
}
