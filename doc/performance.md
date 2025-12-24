# Performance bottlenecks

## Latency incurred by calls from `scpd` into `chuckd`

### Case notes: Single-node benchmark

I discovered that for commit block 10:
- `scheduler_nomination_latency=249.9212ms`
- `nominate_start_to_commit_start_latency=338.218566ms`

The gap is `88.29ms`. Of these `46.248ms` are spent in network calls that `scpd` makes into `chuckd` calling ` scp_handle_get_quorum_set` (which is a function that looks up quorum sets based on their hashes).

The remaining `42.049ms` are unexplained. To explain those, I upgraded chuckd's logging facility to display nanosecond precision, so I'll ask you to rerun the experiment.

The `46.248ms` of calls `scpd`-to-`chuckd` may be hard (but not impossible) to reduce, since scpd is internally sequential. On the upside, these are many calls because with a single node setup the one call to 'scp.nominate' goes through all phases of the consensus protocol. In a real multinode deployment, these calls will be amortized across many slow network calls to other validators and they will have a lower proportional effect. This latency will only disappear completely when chuckd is implemented in rust and the calls between chuckd/scpd are substituted with channel operations within a process. If this latency proves to be problematic in the interim, there is also an intermediate solution, whereby chuckd can proactively push known quorum sets to scpd, so that the lookup for quorum sets is performed from local cache on scpd.

Very cool findings. Let's figure out the remaining `42ms`.
