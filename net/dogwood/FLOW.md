


```

xsend.Send --[transmission-send-workers]-->
    encode
    ssend.Dial
    loop
        ssend.Send --[stream-send-workers]-->
            loop
                caller.Deliver --[VID-call-workers]-->
                    client.Deliver

srecv.Deliver --[stream-receive-workers]-->
    verify
    --chan--> xrecv.do --[transmission-receive-stream-workers]-->
        --[transmission-receive-decode-workers]-->
            decode
    loop
        caller.Deliver --[VID-call-workers]-->
            client.Deliver

```