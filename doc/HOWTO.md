```sh

sed -n 'START,ENDp' file


sed -n '11933,12402p' jack.log
```

- can it be that the coordscheduler_notify_vetted path is clogging the rpc conns
`coordscheduler_notify_vetted; io=1.946710811s, queue=19.16µs, serde=132.211062ms`

- not directly related, but scheduler seems slow on sending vetted txs in this time window:
`coordscheduler_notify_vetted; io=572.680663ms, queue=17.56µs, serde=108.3727ms`
