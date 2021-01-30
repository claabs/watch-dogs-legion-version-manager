package main

import (
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"

	"gopkg.in/yaml.v3"
)

var versions = []string{"1.0.00", "1.0.10", "1.1.00", "1.2.00", "1.2.10", "1.2.20", "1.2.30", "1.2.40", "1.3.00"}
var configPath = path.Join(".", "config.yml")
var gamePath = path.Join(os.Getenv("PROGRAMFILES(X86)"), "Ubisoft", "Ubisoft Game Launcher", "games", "Watch Dogs Legion")
var backupPath = path.Clean(path.Join(gamePath, "..", "Watch Dogs Legion Version Cache"))

func main() {
	// client := resty.New()
	enableUPCAutoUpdates(false)
}

type Config struct {
	currentGameVersion string
	backupPath         string
	gamePath           string
}

func writeDefaultConfig() error {
	cfg := Config{
		currentGameVersion: "1.3.00",
		backupPath:         backupPath,
		gamePath:           gamePath,
	}
	cfgYaml, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(configPath, cfgYaml, 0644)
	if err != nil {
		return err
	}
	return nil
}

func enableUPCAutoUpdates(enabled bool) error {
	// %LOCALAPPDATA%\Ubisoft Game Launcher\settings.yml
	settingsFile := path.Join(os.Getenv("LOCALAPPDATA"), "Ubisoft Game Launcher", "settings.yml")
	data, err := ioutil.ReadFile(settingsFile)
	if err != nil {
		return err
	}

	// Using a real YAML package here forces us to declare a full struct when we unmarshall
	// Since it could change without warning, we just use regex to not break UPC
	// autoPatching.enabled
	autoPatchingRegex := regexp.MustCompile(`autoPatching:[\r\n]{1,2}  enabled: (true|false)`)
	updatedYaml := autoPatchingRegex.ReplaceAllString(string(data), "autoPatching:\r\n  enabled: "+strconv.FormatBool(enabled))
	err = ioutil.WriteFile(settingsFile, []byte(updatedYaml), 0644)
	if err != nil {
		return err
	}
	return nil
}
