package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:generate go get github.com/akavel/rsrc
//go:generate rsrc -manifest wdl-version-manager.exe.manifest -o wdl-version-manager.syso
//go:generate go build -o wdl-version-manager.exe

// var versions = []string{"1.0.00", "1.0.10", "1.1.00", "1.2.00", "1.2.10", "1.2.20", "1.2.30", "1.2.40", "1.3.00"}
var configPath = filepath.Join(".", "config.yml")

func main() {
	// client := resty.New()
	// enableUPCAutoUpdates(false)
	err := cacheGameFiles()
	if err != nil {
		log.Fatalln(err)
	}
}

type Config struct {
	CurrentGameVersion string
	CachePath          string
	GamePath           string
}

func cacheGameFiles() error {
	config, err := getConfig()
	if err != nil {
		return err
	}
	log.Println("game path: " + config.GamePath)
	err = filepath.Walk(config.GamePath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		return moveToCache(filePath, config.GamePath, config.CachePath, config.CurrentGameVersion)
	})
	if err != nil {
		return err
	}
	return err
}

func moveToCache(filePath, gamePath, cachePath, version string) error {
	localPath := getLocalPath(gamePath, filePath)
	cacheFilename := filepath.Join(cachePath, localPath) + "." + version
	log.Println("Copying file " + filePath + " to " + cacheFilename)
	dirPath, _ := filepath.Split(cacheFilename)
	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		return err
	}
	return os.Rename(filePath, cacheFilename)
}

func getLocalPath(gamePath, filePath string) string {
	pathList := strings.Split(filePath, string(os.PathSeparator))
	gamePathList := strings.Split(gamePath, string(os.PathSeparator))
	gamePathLength := len(gamePathList)
	localPathParts := pathList[gamePathLength:]
	return filepath.Join(localPathParts...)
}

func writeDefaultConfig() ([]byte, error) {
	gamePath := filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Ubisoft", "Ubisoft Game Launcher", "games", "Watch Dogs Legion")
	cachePath := filepath.Join(gamePath, "..", "Watch Dogs Legion Version Cache")
	cfg := Config{
		CurrentGameVersion: "1.3.00",
		CachePath:          cachePath,
		GamePath:           gamePath,
	}
	cfgYaml, err := yaml.Marshal(&cfg)
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(configPath, cfgYaml, 0644)
	return cfgYaml, err
}

func getConfig() (*Config, error) {
	log.Println("Getting config...")
	cfgYaml, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Println("Config file doesn't exist, creating one...")
		cfgYaml, err = writeDefaultConfig()
		if err != nil {
			return nil, err
		}
	}
	cfg := &Config{}
	err = yaml.Unmarshal(cfgYaml, &cfg)
	return cfg, err
}

//nolint
func setCurrentGameVersion(version string) error {
	cfg, err := getConfig()
	if err != nil {
		return err
	}
	cfg.CurrentGameVersion = version
	cfgYaml, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(configPath, cfgYaml, 0644)
}

//nolint
func getCurrentGameVersion() (string, error) {
	cfg, err := getConfig()
	if err != nil {
		return "", err
	}
	return cfg.CurrentGameVersion, nil
}

//nolint
func enableUPCAutoUpdates(enabled bool) error {
	// %LOCALAPPDATA%\Ubisoft Game Launcher\settings.yml
	settingsFile := filepath.Join(os.Getenv("LOCALAPPDATA"), "Ubisoft Game Launcher", "settings.yml")
	data, err := ioutil.ReadFile(settingsFile)
	if err != nil {
		return err
	}

	// Using a real YAML package here forces us to declare a full struct when we unmarshall
	// Since it could change without warning, we just use regex to not break UPC
	// autoPatching.enabled
	autoPatchingRegex := regexp.MustCompile(`autoPatching:[\r\n]{1,2}  enabled: (true|false)`)
	updatedYaml := autoPatchingRegex.ReplaceAllString(string(data), "autoPatching:\r\n  enabled: "+strconv.FormatBool(enabled))
	return ioutil.WriteFile(settingsFile, []byte(updatedYaml), 0644)
}
