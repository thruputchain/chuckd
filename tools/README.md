
## Installation on hog cluster

- log into `hog12`
- copy `PKGBUILD` to `/tmp/hogtools` and `cd` there
- run `makepkg -si` (this installs in `/usr/local/bin`)
- run `cd ~dm/ansible_hog` then `sudo ansible-playbook misc-config.yml` to populate to all machines
