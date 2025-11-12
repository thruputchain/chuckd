
## Install and deploy on hog cluster 

### Install `hog_lockd` daemon

- log into `hog12`
- copy `PKGBUILD` to `/tmp/hogtools` and `cd` there
- run `makepkg -si` (this installs in `/usr/local/bin`)
- run `cd ~dm/ansible_hog` then `sudo ansible-playbook misc-config.yml` to populate to all machines

### Deploy `hog_lockd` daemon

