# Analyses

## Timestamp in the future

### Incidence

- coordinator benchmark in cluster: Expected. Hog cluster skew ~20ms.

### Symptom

Coordinator benchmark hangs forever.

Logs have:
```
2025-12-08T15:11:29-08:00 INF /tmp/hogjob-hogci/sub/baton/baton/vconsensus.go:78 cannot validate, timestamp in the future (stamp 2025-12-08T15:11:29.88151065-08:00, now 2025-12-08T15:11:29.8799029-08:00)
```

### Solution

- increase skew parameter in validator config

### Summary

Quorum of validators sees correct timestamp (in the past), whereas a faulty validator outside the quorum sees a timestamp in the future.
Faulty validator is not able to continue participating (hangs forever). Quorum carries on.

### Cause

Caused by time skew between validators.

### Effect

- validator encounters a nomination with a timestamp in the future (possible on hog cluster)
- rejects the only nomination for this round
- the other validators don't observe a timestamp in the future, and externalize the nomination
- rejecting validator keeps receiving scp envelopes which give rise tovalidation requests for following rounds, but
- validator blocks every validation request, waiting to commit the blocks below (which will never happen)
- this blocks the multiplex receive loop (which is waiting on the original envelope to be processes)
- which causes the dogwood broadcast to start dropping receptions
