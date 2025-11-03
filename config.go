package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/jessevdk/go-flags"
	"github.com/zyedidia/eget/home"
)

type ConfigGlobal struct {
	All          bool   `toml:"all"`
	DownloadOnly bool   `toml:"download_only"`
	File         string `toml:"file"`
	GithubToken  string `toml:"github_token"`
	Quiet        bool   `toml:"quiet"`
	ShowHash     bool   `toml:"show_hash"`
	Source       bool   `toml:"download_source"`
	System       string `toml:"system"`
	Target       string `toml:"target"`
	UpgradeOnly  bool   `toml:"upgrade_only"`
}

type ConfigRepository struct {
	All          bool     `toml:"all"`
	AssetFilters []string `toml:"asset_filters"`
	DownloadOnly bool     `toml:"download_only"`
	File         string   `toml:"file"`
	Name         string   `toml:"name"`
	Quiet        bool     `toml:"quiet"`
	ShowHash     bool     `toml:"show_hash"`
	Source       bool     `toml:"download_source"`
	System       string   `toml:"system"`
	Tag          string   `toml:"tag"`
	Target       string   `toml:"target"`
	UpgradeOnly  bool     `toml:"upgrade_only"`
	Verify       string   `toml:"verify_sha256"`
	DisableSSL   bool     `toml:"disable_ssl"`
}

type Config struct {
	Meta struct {
		Keys     []string
		MetaData *toml.MetaData
	}
	Global       ConfigGlobal `toml:"global"`
	Repositories map[string]ConfigRepository
}

func LoadConfigurationFile(path string) (Config, error) {
	var conf Config
	meta, err := toml.DecodeFile(path, &conf)

	if err != nil {
		return conf, err
	}

	meta, err = toml.DecodeFile(path, &conf.Repositories)

	conf.Meta.Keys = make([]string, len(meta.Keys()))

	for i, key := range meta.Keys() {
		conf.Meta.Keys[i] = key.String()
	}

	conf.Meta.MetaData = &meta

	return conf, err
}

func GetOSConfigPath(homePath string) string {
	var configDir string

	defaultConfig := map[string]string{
		"windows": "LocalAppData",
		"default": ".config",
	}

	var goos string
	switch runtime.GOOS {
	case "windows":
		configDir = os.Getenv("LOCALAPPDATA")
		goos = "windows"
	default:
		configDir = os.Getenv("XDG_CONFIG_HOME")
		goos = "default"
	}

	if configDir == "" {
		configDir = filepath.Join(homePath, defaultConfig[goos])
	}

	return filepath.Join(configDir, "eget", "eget.toml")
}

func InitializeConfig() (*Config, error) {
	var err error
	var config Config

	homePath, _ := os.UserHomeDir()
	appName := "eget"

	if configFilePath, ok := os.LookupEnv("EGET_CONFIG"); ok {
		if config, err = LoadConfigurationFile(configFilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s: %w", configFilePath, err)
		}
	} else {
		configFilePath := homePath + "/." + appName + ".toml"
		if config, err = LoadConfigurationFile(configFilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s: %w", configFilePath, err)
		}
	}

	if err != nil {
		configFilePath := appName + ".toml"
		if config, err = LoadConfigurationFile(configFilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s: %w", configFilePath, err)
		}
	}

	configFallBackPath := GetOSConfigPath(homePath)
	if err != nil && configFallBackPath != "" {
		if config, err = LoadConfigurationFile(configFallBackPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s: %w", configFallBackPath, err)
		}
	}

	if err != nil {
		config = Config{
			Global: ConfigGlobal{
				All:          false,
				DownloadOnly: false,
				GithubToken:  "",
				Quiet:        false,
				ShowHash:     false,
				Source:       false,
				UpgradeOnly:  false,
			},
			Repositories: make(map[string]ConfigRepository, 0),
		}

		return &config, nil
	}

	delete(config.Repositories, "global")

	// set default global values
	if !config.Meta.MetaData.IsDefined("global", "all") {
		config.Global.All = false
	}

	if !config.Meta.MetaData.IsDefined("global", "github_token") {
		config.Global.GithubToken = ""
	}

	if !config.Meta.MetaData.IsDefined("global", "quiet") {
		config.Global.Quiet = false
	}

	if !config.Meta.MetaData.IsDefined("global", "download_only") {
		config.Global.DownloadOnly = false
	}

	if !config.Meta.MetaData.IsDefined("global", "show_hash") {
		config.Global.ShowHash = false
	}

	if !config.Meta.MetaData.IsDefined("global", "upgrade_only") {
		config.Global.UpgradeOnly = false
	}

	// set default repository values
	for name, repo := range config.Repositories {

		if !config.Meta.MetaData.IsDefined(name, "all") {
			repo.All = config.Global.All
		}

		if !config.Meta.MetaData.IsDefined(name, "asset_filters") {
			repo.AssetFilters = []string{}
		}

		if !config.Meta.MetaData.IsDefined(name, "download_only") {
			repo.DownloadOnly = config.Global.DownloadOnly
		}

		if !config.Meta.MetaData.IsDefined(name, "quiet") {
			repo.Quiet = config.Global.Quiet
		}

		if !config.Meta.MetaData.IsDefined(name, "show_hash") {
			repo.ShowHash = config.Global.ShowHash
		}

		if !config.Meta.MetaData.IsDefined(name, "target") && config.Global.Target != "" {
			repo.Target = config.Global.Target
		}

		if !config.Meta.MetaData.IsDefined(name, "upgrade_only") {
			repo.UpgradeOnly = config.Global.UpgradeOnly
		}

		if !config.Meta.MetaData.IsDefined(name, "download_source") {
			repo.Source = config.Global.Source
		}

		config.Repositories[name] = repo
	}

	return &config, nil
}

func update[T any](config T, cli *T) T {
	if cli == nil {
		return config
	}
	return *cli
}

// substituteTemplateVars replaces {{.OS}} and {{.Arch}} in asset filter strings
// with the actual OS and architecture values based on the system parameter.
func substituteTemplateVars(filter string, system string) string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// If system is specified, parse it to extract OS and Arch
	if system != "" && system != "all" {
		parts := strings.Split(system, "/")
		if len(parts) >= 2 {
			goos = parts[0]
			goarch = parts[1]
		}
	}

	result := strings.ReplaceAll(filter, "{{.OS}}", goos)
	result = strings.ReplaceAll(result, "{{.Arch}}", goarch)
	return result
}

// Move the loaded configuration file global options into the opts variable
func SetGlobalOptionsFromConfig(config *Config, parser *flags.Parser, opts *Flags, cli CliFlags) error {
	if config.Global.GithubToken != "" && os.Getenv("EGET_GITHUB_TOKEN") == "" {
		os.Setenv("EGET_GITHUB_TOKEN", config.Global.GithubToken)
	}

	opts.Tag = update("", cli.Tag)
	opts.Prerelease = update(false, cli.Prerelease)
	opts.Source = update(config.Global.Source, cli.Source)
	targ := update(config.Global.Target, cli.Output)
	expanded, err := home.Expand(targ)
	if err != nil {
		return err
	}
	opts.Output = expanded
	opts.System = update(config.Global.System, cli.System)
	opts.ExtractFile = update("", cli.ExtractFile)
	opts.All = update(config.Global.All, cli.All)
	opts.Quiet = update(config.Global.Quiet, cli.Quiet)
	opts.DLOnly = update(config.Global.DownloadOnly, cli.DLOnly)
	opts.UpgradeOnly = update(config.Global.UpgradeOnly, cli.UpgradeOnly)
	opts.Asset = update([]string{}, cli.Asset)
	opts.Hash = update(config.Global.ShowHash, cli.Hash)
	opts.Verify = update("", cli.Verify)
	opts.Remove = update(false, cli.Remove)
	opts.DisableSSL = update(false, cli.DisableSSL)
	return nil
}

// Move the loaded configuration file project options into the opts variable
func SetProjectOptionsFromConfig(config *Config, parser *flags.Parser, opts *Flags, cli CliFlags, projectName string) error {
	for name, repo := range config.Repositories {
		if name == projectName {
			opts.All = update(repo.All, cli.All)
			opts.DLOnly = update(repo.DownloadOnly, cli.DLOnly)
			opts.ExtractFile = update(repo.File, cli.ExtractFile)
			opts.Hash = update(repo.ShowHash, cli.Hash)
			targ, err := home.Expand(repo.Target)
			if err != nil {
				return err
			}
			opts.Output = update(targ, cli.Output)
			opts.Quiet = update(repo.Quiet, cli.Quiet)
			opts.Source = update(repo.Source, cli.Source)
			opts.System = update(repo.System, cli.System)
			opts.Tag = update(repo.Tag, cli.Tag)
			opts.UpgradeOnly = update(repo.UpgradeOnly, cli.UpgradeOnly)
			opts.Verify = update(repo.Verify, cli.Verify)
			opts.DisableSSL = update(repo.DisableSSL, cli.DisableSSL)

			// Apply template substitution to asset_filters
			// We need to determine the system value to use for substitution
			systemForTemplate := opts.System
			if systemForTemplate == "" {
				systemForTemplate = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
			}

			assetFilters := repo.AssetFilters
			if cli.Asset == nil && len(assetFilters) > 0 {
				// Only apply template substitution if we're using config asset_filters
				substitutedFilters := make([]string, len(assetFilters))
				for i, filter := range assetFilters {
					substitutedFilters[i] = substituteTemplateVars(filter, systemForTemplate)
				}
				opts.Asset = substitutedFilters
			} else {
				opts.Asset = update(assetFilters, cli.Asset)
			}

			break
		}
	}
	return nil
}
