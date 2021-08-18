---
title: get
section: 1
header: Get Manual
---

# NAME
  Get - Easily install prebuilt binaries from GitHub

# SYNOPSIS
  get `[--version] [--help] [OPTIONS] REPO`

# DESCRIPTION
  Get is a tool for downloading and extracting prebuilt binaries from releases
  on GitHub. To use it, provide a repository and `get` will search through the
  assets from the latest release in an attempt to find a suitable prebuilt
  binary for your system. If one is found, the asset will be downloaded and
  `get` will extract the binary to the current directory. Get should only be
  used for installing simple, static prebuilt binaries, where the extracted
  binary is all that is needed for installation. For more complex installation,
  you may use the `--download-only` option, and perform extraction manually.

  The behavior of get is configurable in a number of ways via options.
  Documentation for these options is provided below.

# OPTIONS
  `-t, --tag=`

:    Use the given tagged release instead of the latest release. Example: **`get -t nightly zyedidia/micro`**.

  `--to=`

:    Extract the executable to the given directory. Example: **`get zyedidia/micro --to /usr/local/bin`**.

  `-s, --system=`

:    Use the given system as the target instead of the host. Systems follow the notation 'OS/Arch', where OS is a valid OS (darwin, windows, linux, netbsd, openbsd, freebsd, android, illumos, solaris, plan9), and Arch is a valid architecture (amd64, 386, arm, arm64, riscv64). If the special value **all** is used, all possibilities are given and the user must select manually. Example: **`get -s darwin/amd64 zyedidia/micro`**.

  `-f, --file=`

:    Extract the file with the given filename. You may want use this option to extract non-binary files. Example: **`get -f LICENSE zyedidia/micro`**.

  `-q, --quiet`

:    Only print essential output.

  `--download-only`

:    Stop after downloading the asset. This prevents get from performing extraction, allowing you to perform manual installation after the asset is downloaded.

  `--url`

:    Download using a direct URL rather than auto-detecting a release from GitHub. Example **`get --url https://github.com/zyedidia/micro/releases/download/v2.0.10/micro-2.0.10-linux64.tar.gz`**.

  `--asset=`

:    Download a specific asset. Example: **`get --asset nvim.appimage neovim/neovim`**.

  `--rename=`

:    Rename extracted file to given name. Example: **`get --asset nvim.appimage --rename nvim neovim/neovim`**.

  `-x`

:    Force the extracted file to be executable. Example: **`get -x --asset nvim.appimage neovim/neovim`**.

  `--sha256`

:    Show the SHA-256 hash of the downloaded asset. This can be used to verify that the asset is not corrupted.

  `-v, --version`

:    Show version information.

  `-h, --help`

:    Show a help message.

# BUGS

See GitHub Issues: <https://github.com/zyedidia/get/issues>

# AUTHOR

Zachary Yedidia <zyedidia@gmail.com>
