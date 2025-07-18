


```

xsend.Send --[transmission-send-workers]--> 30msec
    encode
    ssend.Dial
    loop
        ssend.Send --[stream-send-workers]--> 10msec
            loop
                caller.Deliver --[VID-call-workers]--> 50msec
                    client.Deliver

srecv.Deliver --[stream-receive-workers]--> 2msec
    verify
    --chan--> xrecv.do --[transmission-receive-stream-workers]-->
        --[transmission-receive-decode-workers]--> 1msec
            decode
    loop
        caller.Deliver --[VID-call-workers]--> 50msec
            client.Deliver

```