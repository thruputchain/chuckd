
## ALGORITHM

- run `ssh-add -L`

- extract (public key, associated git url)

  - create `~/.ssh/key-[sha256(public key)]`

  - append to `~/.gitconfig`:
```
[url "git@key-key-[sha256(public key)].github.com:groundhogchain/xdr_common"]
	insteadOf = https://github.com/groundhogchain/xdr_common
	insteadOf = git@github.com:groundhogchain/xdr_common
	insteadOf = ssh://git@github.com/groundhogchain/xdr_common
```

  - append to `~/.ssh/config`:
```
Host key-[sha256(public key)].github.com
    HostName github.com
    IdentityFile /home/hogci/.ssh/key-[sha256(public key)]
    IdentitiesOnly yes
```




## SCRAP

- `ssh-add -L` outputs:

```
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG5kJv7W0rGMFpjusIyyuwncFsgvlKGRDfiHTqC01bbJ git@github.com:groundhogchain/chuckd.git
```

- append to `.gitconfig`:

```
[url "git@key-9804a9515d0f188e904ebfae72108c659015fbde6c1ed35c5628e0ee2a352d29.github.com:groundhogchain/xdr_common"]
	insteadOf = https://github.com/groundhogchain/xdr_common
	insteadOf = git@github.com:groundhogchain/xdr_common
	insteadOf = ssh://git@github.com/groundhogchain/xdr_common
```

- append to `.ssh/config`:

```
Host key-98141cd755f98fc1dd96935794e5803f471a54c6bab2637acb2aa4bad7dd6f7d.github.com
    HostName github.com
    IdentityFile /home/hogci/.ssh/key-98141cd755f98fc1dd96935794e5803f471a54c6bab2637acb2aa4bad7dd6f7d
    IdentitiesOnly yes
```
