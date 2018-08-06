# U2F Lock

This small application provides an easy way to lock and unlock your i3lock session using a FIDO U2F Token (Yubico Security Key, Google Titan Security Key).

## Features

* Register multiple Keys
* Unlock (kill) i3lock by connecting and activating your Key
* Lock your device with custom command when disconnecting a known Key (default: systemctl suspend)

## Usage

### Install

```bash
go get -u github.com/kwiesmueller/u2flock
```

### Preparing

The Token registration handles get stored in a json file. Always override the default value.

```bash
u2flock -register -key=/home/yourUsername/.secret/u2flock-key.json
```

Touch your token when being asked.

### Unlocking

Add the following snippet to your lock.sh right after the i3lock call:

```bash
U2FLOCKKEY=$HOME/.secret/u2flock-key.json
$HOME/go/bin/u2flock -key=$U2FLOCKKEY &
```

Now when locking you should be able to connect your Key and press the button to unlock.

### Locking

To autolock when disconnecting a known token, add the following snippet to your xinitrc file:

```bash
u2flock -watch -lockCmd=$HOME/.config/i3/lock.sh -lockArgs="" &
```

Not setting `-lockCmd` and `-lockArgs` will suspend instead of locking.

## Disclaimer

This project is built for private purposes, barely tested and probably not secure.
U2F should be used as a 2nd factor rather than this.

The author is not liable for any harms caused by using this software.

A License can be found in [LICENSE](LICENSE).