# Get: easy pre-built binary installation

Get is a tool for downloading and extracting prebuilt binaries from releases
on GitHub. To use it, provide a repository and `get` will search through the
assets from the latest release in an attempt to find a suitable prebuilt
binary for your system. If one is found, the asset will be downloaded and
`get` will extract the binary to the current directory. Get should only be
used for installing simple, static prebuilt binaries, where the extracted
binary is all that is needed for installation. For more complex installation,
you may use the `--download-only` option, and perform extraction manually.

For software maintainers, if you provide prebuilt binaries on GitHub, you can list `get`
as a one-line method for users to install your software.

# How to get get

Before you can get anything, you have to get get.

```
$ go get github.com/zyedidia/get
```

Pre-built binaries and a quick install script coming soon!

# Options

```
Usage:
  get [OPTIONS] REPO

Application Options:
  -t, --tag=           tagged release to use instead of latest
      --to=            extract to directory
  -y                   automatically approve all yes/no prompts
  -s, --system=        target system to download for
  -f, --file=          file name to extract
  -q, --quiet          only print essential output
      --download-only  stop after downloading the asset (no extraction)
      --url            download from the given URL directly
      --asset=         download a specific asset
  -x                   force the extracted file to be executable
      --sha256         show the SHA-256 hash of the downloaded asset
  -v, --version        show version information
  -h, --help           Show this help message
```

# Examples

```
$ get zyedidia/micro
$ get jgm/pandoc
$ get junegunn/fzf
$ get -x --asset nvim.appimage neovim/neovim
$ get zachjs/sv2v
$ get ogham/exa
$ get sharkdp/fd
$ get BurntSushi/ripgrep
```
