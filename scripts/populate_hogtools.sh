#!/bin/sh

hosts=("hog1" "hog2" "hog3" "hog4" "hog5" "hog6" "hog7" "hog8" "hog9" "hog10" "hog11")

user="$(whoami)"
content="lockd=\"hog12:60006\"
job_dir=\"/tmp/hogjob-$user\"
"

for host in "${hosts[@]}"; do
echo "Updating ~/.hogtools on $host..."
printf "%s" "$content" | ssh "$host" "cat > ~/.hogtools"
done
