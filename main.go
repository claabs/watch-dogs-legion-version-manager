package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/go-resty/resty/v2"
	"gopkg.in/yaml.v3"
)

//go:generate go get github.com/akavel/rsrc
//go:generate rsrc -manifest wdl-version-manager.exe.manifest -o wdl-version-manager.syso
//go:generate go build -o wdl-version-manager.exe

type Config struct {
	CurrentGameVersion string
	CachePath          string
	GamePath           string
}

var fileServerRoot = "https://wdlpatches2.charlielaabs.com"
var archiveUser = os.Getenv("ARCHIVE_USER")
var archivePass = os.Getenv("ARCHIVE_PASS")

var ignoredGameDirs = []string{"logs", "Support", filepath.Join("bin", "BattlEye", "Privacy"), filepath.Join("bin", "logs")}
var configPath = filepath.Join(".", "config.yml")

func main() {
	versions, err := getVersions()
	if err != nil {
		fmt.Println(err.Error())
	}
	// enableUPCAutoUpdates(false)
	version, err := getCurrentGameVersion()
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println("Current version: " + version)
	var qs = []*survey.Question{
		{
			Name: "desiredVersion",
			Prompt: &survey.Select{
				Message: "Select a version to switch to:",
				Options: versions,
			},
		},
	}
	answers := struct {
		DesiredVersion string
	}{}

	// perform the questions
	err = survey.Ask(qs, &answers)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("Desired version: " + answers.DesiredVersion)
	// err = cacheGameFiles()
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println("Press Enter to exit")
	fmt.Scanln()
}

func getClient() *resty.Client {
	client := resty.New()
	client.SetBasicAuth(archiveUser, archivePass)
	client.SetHostURL(fileServerRoot)
	return client
}

func getVersions() ([]string, error) {
	client := getClient()
	resp, err := client.R().Get("versions.txt")
	if err != nil || resp.StatusCode() != 200 {
		return nil, err
	}
	versions := strings.Split(string(resp.Body()), "\n")
	return versions, nil
}

func latestFileForVersion(verion string, versions []string) (string, error) {
	return "", nil
}

func cacheGameFiles() error {
	config, err := getConfig()
	if err != nil {
		return err
	}
	fmt.Println("game path: " + config.GamePath)
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
	for _, value := range ignoredGameDirs {
		if localPath[:len(value)] == value {
			fmt.Println("Ignoring file in \"" + value + "\" folder")
			return nil
		}
	}
	cacheFilename := filepath.Join(cachePath, localPath) + "." + version
	fmt.Println("Copying file " + filePath + " to " + cacheFilename)
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
	cfgYaml, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Println("Config file doesn't exist, creating one...")
		cfgYaml, err = writeDefaultConfig()
		if err != nil {
			return nil, err
		}
	}
	cfg := &Config{}
	err = yaml.Unmarshal(cfgYaml, &cfg)
	return cfg, err
}

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

func getCurrentGameVersion() (string, error) {
	cfg, err := getConfig()
	if err != nil {
		return "", err
	}
	return cfg.CurrentGameVersion, nil
}

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
