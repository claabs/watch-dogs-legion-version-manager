package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/go-resty/resty/v2"
	"gopkg.in/yaml.v3"
)

//go:generate go get github.com/akavel/rsrc
//go:generate rsrc -manifest wdl-version-manager.exe.manifest -o wdl-version-manager.syso
//go:generate go build -o wdl-version-manager.exe

// To run locally: ARCHIVE_USER=username ARCHIVE_PASS=password ./wdl-version-manager.exe

type Config struct {
	CurrentGameVersion string
	CachePath          string
	GamePath           string
	SavePath           string
}

var fileServerRoot = "https://wdlpatches2.charlielaabs.com"
var gameId = "3353"
var archiveUser = os.Getenv("ARCHIVE_USER")
var archivePass = os.Getenv("ARCHIVE_PASS")

var ignoredGameDirs = []string{"logs", "Support", filepath.Join("bin", "BattlEye", "Privacy"), filepath.Join("bin", "logs")}
var configPath = filepath.Join(".", "config.yml")

func handleError(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	fmt.Println("Press [ENTER] to exit...")
	fmt.Scanln()
	os.Exit(1)
}

func main() {
	versions, err := getVersions()
	if err != nil {
		handleError(err)
	}
	// enableUPCAutoUpdates(false)
	version, err := getCurrentGameVersion()
	if err != nil {
		handleError(err)
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
	err = survey.Ask(qs, &answers)
	if err != nil {
		handleError(err)
	}
	desiredVersion := answers.DesiredVersion
	fmt.Println("Desired version: " + desiredVersion)
	downgrade := isDowngrade(desiredVersion, versions)

	printUPCReminder(downgrade)
	err = setUPCAutoUpdate(downgrade)
	if err != nil {
		handleError(err)
	}

	// err = cacheGameFiles()
	// if err != nil {
	// 	handleError(err)
	// }

	err = backupSaves(desiredVersion)
	if err != nil {
		handleError(err)
	}
	err = setCurrentGameVersion(desiredVersion)
	if err != nil {
		handleError(err)
	}

	fmt.Println("Press [ENTER] to exit...")
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
	if err != nil {
		return nil, err
	}
	statusCode := resp.StatusCode()
	if statusCode != 200 {
		return nil, errors.New("Status code: " + resp.Status())
	}
	versions := strings.Split(string(resp.Body()), "\r\n")
	return versions, nil
}

func latestFileForVersion(filename, desiredVersion string, versions []string) (string, error) {

	// Todo: slice version range
	var versionIdx int
	for i, version := range versions {
		if version == desiredVersion {
			versionIdx = i
			break
		}
	}

	versions = versions[:versionIdx+1]
	existsList := make([]bool, len(versions))
	fatalErrors := make(chan error)
	wgDone := make(chan bool)
	wg := sync.WaitGroup{}

	fmt.Println("Versions to fetch: " + strings.Join(versions, ", "))

	for i, version := range versions {
		wg.Add(1)
		go func(i int, version string) {
			exists, err := remoteFileVersionExists(filename, version)
			if err != nil {
				fatalErrors <- err
				wg.Done()
				return
			}

			existsList[i] = exists
			fmt.Println("Fetched version " + filename + "." + version + " and got exists: " + strconv.FormatBool(exists))
			wg.Done()
		}(i, version)
	}

	go func() {
		wg.Wait()
		close(wgDone)
	}()

	select {
	case <-wgDone:
		break
	case err := <-fatalErrors:
		// close(fatalErrors)
		return "", err
	}

	fmt.Println("Exists results: ")

	last := len(existsList) - 1
	latestVersion := ""
	for i := range existsList {
		if existsList[last-i] {
			latestVersion = versions[last-i]
			break
		}
	}
	if latestVersion == "" {
		return "", errors.New("No valid versions exist for file " + filename + " and desired version " + desiredVersion)
	}
	return filename + "." + latestVersion, nil
}

func remoteFileVersionExists(filename, version string) (bool, error) {
	client := getClient()
	path := filename + "." + version
	resp, err := client.R().Head(path)
	if err != nil {
		return false, err
	}
	if resp.StatusCode() != 200 {
		return false, nil
	}
	return true, nil
}

func obtainFile(filenameWithVersion string) error {
	config, err := getConfig()
	if err != nil {
		return err
	}

	filename := filenameWithVersion[:len(filenameWithVersion)-8] // Remove 7 character version suffix
	outputPath := filepath.Join(config.GamePath, filename)
	cachePath := filepath.Join(config.CachePath, filenameWithVersion)

	info, err := os.Stat(cachePath)
	if info.IsDir() {
		return errors.New("Cannot obtain a file that is a directory")
	}
	if os.IsNotExist(err) {
		return downloadRemoteFile(filenameWithVersion, outputPath)
	}
	return moveFileFromCache(cachePath, outputPath)
}

// Download an individual file and place it in the game directory with its original version name
// The files in the game directory should be cached before performing this
func downloadRemoteFile(filenameWithVersion, outputPath string) error {
	client := getClient()
	resp, err := client.R().SetOutput(outputPath).Get(filenameWithVersion)
	if err != nil {
		return err
	}
	if resp.StatusCode() != 200 {
		return errors.New("Unable to download file " + filenameWithVersion + " with status " + resp.Status())
	}
	return nil
}

// Get a file from the cache and place it in the game directory with its original version name
// The files in the game directory should be cached before performing this
func moveFileFromCache(cachePath, outputPath string) error {
	err := os.Rename(cachePath, outputPath)
	if err != nil {
		return err
	}
	return nil
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

func getSavePath() (string, error) {
	savegamesRoot := filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Ubisoft", "Ubisoft Game Launcher", "savegames")
	infos, err := ioutil.ReadDir(savegamesRoot)
	if err != nil {
		return "", err
	}
	if len(infos) != 1 {
		return "", errors.New("Unable to find one user save folder")
	}
	userId := infos[0].Name()
	savegamesDir := filepath.Join(savegamesRoot, userId, gameId)
	return savegamesDir, nil
}

func backupSaves(version string) error {
	fmt.Println("Backing up save files...")
	config, err := getConfig()
	if err != nil {
		return err
	}
	saveFiles := []string{"1.save", "2.save", "3.save", "4.save"}
	for _, saveFile := range saveFiles {
		oldFilePath := filepath.Join(config.SavePath, saveFile)
		newFileName := saveFile + "." + version + "." + time.Now().Format(time.RFC3339) + ".bak"
		newFilePath := filepath.Join(config.SavePath, newFileName)
		err = os.Rename(oldFilePath, newFilePath)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeDefaultConfig() ([]byte, error) {
	gamePath := filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Ubisoft", "Ubisoft Game Launcher", "games", "Watch Dogs Legion")
	cachePath := filepath.Join(gamePath, "..", "Watch Dogs Legion Version Cache")
	savePath, saveErr := getSavePath()
	cfg := Config{
		CurrentGameVersion: "1.3.00", // TODO: Check server for latest version
		CachePath:          cachePath,
		GamePath:           gamePath,
		SavePath:           savePath,
	}
	cfgYaml, err := yaml.Marshal(&cfg)
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(configPath, cfgYaml, 0644)
	if err != nil {
		return nil, err
	}
	if saveErr != nil {
		return nil, errors.New("Unable to autoomatically detect save file location. Please set manually in in config.yml")
	}
	return cfgYaml, nil
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

func setUPCAutoUpdate(isDowngrade bool) error {
	if !isDowngrade {
		fmt.Println("Enabling Ubisoft Connect auto updates...")
		return enableUPCAutoUpdates(true)
	}
	fmt.Println("Disabling Ubisoft Connect auto updates...")
	return enableUPCAutoUpdates(false)
}

func enableUPCAutoUpdates(enabled bool) error {
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

func printUPCReminder(isDowngrade bool) {
	if isDowngrade {
		fmt.Println("Before continuing, ensure Ubisoft Connect is open, and in online mode.")
		fmt.Println("Also make sure the game is in your desired video mode (DX11/DX12), as changing it will invalidate the DRM and require a temporary revert to the latest version.")
		fmt.Println("Press [ENTER] to continue...")
		fmt.Scanln()
	}
}

func isDowngrade(desiredVersion string, versions []string) bool {
	latestVersion := versions[len(versions)-1]
	return desiredVersion != latestVersion
}
