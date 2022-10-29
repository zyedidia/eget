package main

import (
	"os"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/jessevdk/go-flags"
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

func InitializeConfig() (*Config, error) {
	homePath, _ := os.UserHomeDir()
	appPath := path.Dir(os.Args[0])
	appName := path.Base(os.Args[0])

	config, err := LoadConfigurationFile(homePath + "/." + appName + ".toml")

	if err != nil {
		config, err = LoadConfigurationFile(appPath + "/." + appName + ".toml")
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

	if !config.Meta.MetaData.IsDefined("global", "target") {
		cwd, _ := os.Getwd()
		config.Global.Target = cwd
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

		if !config.Meta.MetaData.IsDefined(name, "target") {
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

	return &config, err
}

// Move the loaded configuration file options into the opts variable
func SetOptionsFromConfig(config *Config, parser *flags.Parser, opts *Flags, projectName string) {

	if config.Global.GithubToken != "" && os.Getenv("EGET_GITHUB_TOKEN") == "" {
		os.Setenv("EGET_GITHUB_TOKEN", config.Global.GithubToken)
	}

	opts.All = config.Global.All
	opts.DLOnly = config.Global.DownloadOnly
	opts.Hash = config.Global.ShowHash
	opts.Output = config.Global.Target
	opts.Quiet = config.Global.Quiet
	opts.System = config.Global.System
	opts.UpgradeOnly = config.Global.UpgradeOnly
	opts.Source = config.Global.Source

	for name, repo := range config.Repositories {
		if name == projectName {
			opts.All = repo.All
			opts.Asset = repo.AssetFilters
			opts.DLOnly = repo.DownloadOnly
			opts.ExtractFile = repo.File
			opts.Hash = repo.ShowHash
			opts.Output = repo.Target
			opts.Quiet = repo.Quiet
			opts.Source = repo.Source
			opts.System = repo.System
			opts.Tag = repo.Tag
			opts.UpgradeOnly = repo.UpgradeOnly
			break
		}
	}
}
