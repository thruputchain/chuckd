

```sh

# acquire and hold locks hog1 and hog2 until killed
hog_hold_lock —j=id --hosts=hog1,hog2 &

# wait until locks hog1 and hog2 are acquired
hog_wait_lock —j=id --hosts=hog1,hog2

# execute the script on stdin on hog1 in the directory of job id
hog_exec —j=id —host=hog1 <EOF
  ...
EOF

# 
hog_copy —j=id \
  —in=hog1 —out=hog2 \
  —in=filepath1 —out=filepath1 \
  —in=filepath2 —out=filepath2

```
