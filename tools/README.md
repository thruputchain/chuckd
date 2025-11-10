

```sh

# acquire and hold locks hog1 and hog2 until killed
hog_hold_lock —j=id --host=hog1 --host=hog2 &

# wait until locks hog1 and hog2 are acquired
hog_wait_lock —j=id --host=hog1 --host=hog2

# execute the script on stdin on hog1 in the directory of job id
hog_exec —j=id —host=hog1 <EOF
  ...
EOF

# copy path1 and path3 from the job id directory on hog1 to
# path2 and path4 in the job id directory on hog2
hog_copy —j=id \
  —in=hog1 —out=hog2 \
  —in=path1 —out=path2 \
  —in=path3 —out=path4
```
