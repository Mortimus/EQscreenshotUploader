package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml"
)

var configuration Configuration

var configPath = "config.toml"

// Configured in secrets.go init()
var DiscordToken = ""
var DiscordChannel = ""

func init() {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	configuration, err = loadConfig(filepath.Join(exPath, configPath))
	if err != nil {
		configuration, err = loadConfig(configPath)
		if err != nil {
			err = ioutil.WriteFile(configPath, nil, 0644)
			if err != nil {
				log.Printf("Error writing config: %s", err.Error())
			}
		}
	} else {
		configPath = filepath.Join(exPath, configPath)
	}
	loadDefaults()                 // check for missing values, and set defaults
	configuration.Save(configPath) // save config to write the file if missing
}

type Main struct {
	UploadDelay         int     `comment:"Time to wait (in seconds) to upload the screenshot. We have to delay until it's finished writing and released for reading defaults to 5 seconds"`
	ScreenshotExtension string  `comment:"Extension for screenshots defaults to .jpg"`
	BlurPartial         bool    `comment:"Blur partial screenshots. Defaults to false"`
	BlurAmount          float64 `comment:"Amount of blur to apply to partial screenshots. Defaults to 6.5"`
	BlurXStart          int     `comment:"X start coordinate of box to blur"`
	BlurYStart          int     `comment:"y start coordinate of box to blur"`
	BlurXEnd            int     `comment:"X end coordinate of box to blur"`
	BlurYEnd            int     `comment:"y end coordinate of box to blur"`
}

type Discord struct {
	Token     string `comment:"Discord Bot Token leave blank to use built in server"`
	ChannelID string `comment:"Discord Channel to sent screenshots to leave blank to use built in server"`
}

type Everquest struct {
	ScreenshotPath string `comment:"Path to screenshot folder leave blank to auto detect"`
}

type Log struct {
	Path string `comment:"Where to store logs leave blank to use EQscreenshotUploader.log"`
}

type Configuration struct {
	Main      Main
	Everquest Everquest
	Log       Log
	Discord   Discord
}

func loadConfig(path string) (Configuration, error) {
	config := Configuration{}
	configFile, err := ioutil.ReadFile(path)
	if err != nil {
		return config, err
	}
	err = toml.Unmarshal(configFile, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

func loadDefaults() {
	if configuration.Main.UploadDelay <= 0 {
		configuration.Main.UploadDelay = 5
	}
	if configuration.Main.ScreenshotExtension == "" {
		configuration.Main.ScreenshotExtension = ".jpg"
	}
	if configuration.Everquest.ScreenshotPath == "" {
		configuration.Everquest.ScreenshotPath = findPath()
	}
	if configuration.Log.Path == "" {
		configuration.Log.Path = "EQscreenshotUploader.log"
	}
	if configuration.Main.BlurAmount == 0 {
		configuration.Main.BlurAmount = 6.5
	}
}

func (c Configuration) Save(path string) {
	out, err := toml.Marshal(c)
	if err != nil {
		log.Printf("Error marshalling config: %s", err.Error())
	}
	err = ioutil.WriteFile(path, out, 0644)
	if err != nil {
		log.Printf("Error writing config: %s", err.Error())
	}
}

func getDiscordToken() string { // Helper function to use secrets.go if not overridden
	if configuration.Discord.Token == "" {
		return DiscordToken
	}
	return configuration.Discord.Token
}

func getDiscordChannel() string { // Helper function to use secrets.go if not overridden
	if configuration.Discord.ChannelID == "" {
		return DiscordChannel
	}
	return configuration.Discord.ChannelID
}
