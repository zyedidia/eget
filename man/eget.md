---
title: eget
section: 1
header: Eget Manual
---

# NAME
  eget - easily install prebuilt binaries from GitHub

# SYNOPSIS
  eget `[--version] [--help] [OPTIONS] PROJECT`

# DESCRIPTION
  Eget is a tool for downloading and extracting prebuilt binaries from releases
  on GitHub. To use it, provide a repository and Eget will search through the
  assets from the latest release in an attempt to find a suitable prebuilt
  binary for your system. If one is found, the asset will be downloaded and
  Eget will extract the binary to the current directory. Eget should only be
  used for installing simple, static prebuilt binaries, where the extracted
  binary is all that is needed for installation. For more complex installation,
  you may use the `--download-only` option, and perform extraction manually.

  The **`PROJECT`** argument passed to Eget should either be a GitHub
  repository, formatted as **`user/repo`**, in which case Eget will search the
  release assets, or a direct URL, in which case Eget will directly download
  and extract from the given URL.

  If Eget downloads an asset called `xxx` and there also exists an asset called
  `xxx.sha256` or `xxx.sha256sum`, Eget will automatically verify that the
  SHA-256 checksum of the downloaded asset matches the one contained in that
  file, and abort installation if a mismatch occurs.

  The behavior of Eget is configurable in a number of ways via options.
  Documentation for these options is provided below.

# OPTIONS
  `-t, --tag=`

:    Use the given tagged release instead of the latest release. Example: **`eget -t nightly zyedidia/micro`**.

  `--pre-release`

:    Include pre-releases when fetching the latest version. This will get the latest overall release, even if it is a pre-release.

  `--to=`

:    Move the executable to the given name after extraction. Example: **`eget zyedidia/micro --to /usr/local/bin`**. Example: **`eget --asset nvim.appimage --to nvim neovim/neovim`**.

  `-s, --system=`

:    Use the given system as the target instead of the host. Systems follow the notation 'OS/Arch', where OS is a valid OS (darwin, windows, linux, netbsd, openbsd, freebsd, android, illumos, solaris, plan9), and Arch is a valid architecture (amd64, 386, arm, arm64, riscv64). If the special value **all** is used, all possibilities are given and the user must select manually. Example: **`eget -s darwin/amd64 zyedidia/micro`**.

  `-f, --file=`

:    Extract the file with the given filename. You may want use this option to extract non-binary files. Example: **`eget -f LICENSE zyedidia/micro`**.

  `-q, --quiet`

:    Only print essential output.

  `--download-only`

:    Stop after downloading the asset. This prevents Eget from performing extraction, allowing you to perform manual installation after the asset is downloaded.

  `--asset=`

:    Download a specific asset containing the given string. If there is an exact match with an asset, that asset is used regardless. Example: **`eget --asset nvim.appimage neovim/neovim`**.

  `--sha256`

:    Show the SHA-256 hash of the downloaded asset. This can be used to verify that the asset is not corrupted.

  `--verify-sha256=`

:    Verify the SHA-256 hash of the downloaded asset against the one provided as an argument. Similar to `--sha256`, but Eget will do the verification for you.

  `-v, --version`

:    Show version information.

  `-h, --help`

:    Show a help message.

# BUGS

See GitHub Issues: <https://github.com/zyedidia/eget/issues>

# AUTHOR

Zachary Yedidia <zyedidia@gmail.com>
