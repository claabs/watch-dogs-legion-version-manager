package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/go-resty/resty/v2"
	"github.com/melbahja/got"
	"github.com/vbauerster/mpb/v6"
	"github.com/vbauerster/mpb/v6/decor"
	"golang.org/x/term"
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
var archiveUserPack string
var archivePassPack string
var archiveUserEnv = os.Getenv("ARCHIVE_USER")
var archivePassEnv = os.Getenv("ARCHIVE_PASS")
var archiveUser = checkEmptyString(archiveUserEnv, archiveUserPack)
var archivePass = checkEmptyString(archivePassEnv, archivePassPack)

var ignoredGameDirs = []string{
	"logs",
	"Support",
	filepath.Join("bin", "BattlEye", "Privacy"),
	filepath.Join("bin", "logs"),
	"uplay_install.state",
}
var configPath = filepath.Join(".", "config.yml")
var config = &Config{}

func checkEmptyString(potEmpty, defaultVal string) string {
	if potEmpty == "" {
		return defaultVal
	}
	return potEmpty
}

func handleError(err error) {
	fmt.Fprintln(os.Stderr, "Error: "+err.Error())
	fmt.Println("Press [ENTER] to exit...")
	fmt.Scanln()
	os.Exit(1)
}

func main() {
	versions, err := getVersions()
	if err != nil {
		handleError(err)
	}
	config, err = getConfig(versions)
	if err != nil {
		handleError(err)
	}
	// enableUPCAutoUpdates(false)
	version := config.CurrentGameVersion
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

	err = versionChangeAllFiles(desiredVersion, versions)
	if err != nil {
		handleError(err)
	}

	err = backupSaves(desiredVersion)
	if err != nil {
		handleError(err)
	}

	err = setCurrentGameVersion(desiredVersion)
	if err != nil {
		handleError(err)
	}

	fmt.Println("Congrats! Your game files have been changed to version " + desiredVersion)
	fmt.Println("Next, start the game once with Ubisoft Connect in online mode")
	fmt.Println("Then close the game, switch to offline mode in Ubisoft Connect, and start the game again")

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

func versionChangeAllFiles(desiredVersion string, versions []string) error {
	movableFiles := []string{}
	err := filepath.Walk(config.GamePath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		localFilePath := getLocalPath(filePath)
		if isIgnoredFile(localFilePath) {
			return nil
		}
		movableFiles = append(movableFiles, localFilePath)
		return nil
	})
	if err != nil {
		return err
	}

	fatalErrors := make(chan error)
	wgDone := make(chan bool)
	wg := sync.WaitGroup{}
	multiProgress := mpb.New(mpb.WithWidth(getWidth())) // , mpb.WithWaitGroup(&wg)

	fmt.Println("Files to change: " + strings.Join(movableFiles, ", "))

	for _, filename := range movableFiles {
		wg.Add(1)
		go func(filename, desiredVersion string, versions []string, multiProgress *mpb.Progress) {
			err := versionChangeFile(filename, desiredVersion, versions, multiProgress)
			if err != nil {
				fatalErrors <- err
			}
			wg.Done()
		}(filename, desiredVersion, versions, multiProgress)
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
		return err
	}

	return nil
}

func versionChangeFile(filename, desiredVersion string, versions []string, multiProgress *mpb.Progress) error {
	resolvedCurrentVersionFilePath, err := latestFileForVersion(filename, config.CurrentGameVersion, versions)
	if err != nil {
		return err
	}
	resolvedDesiredVersionFilePath, err := latestFileForVersion(filename, desiredVersion, versions)
	if err != nil {
		return err
	}

	if resolvedCurrentVersionFilePath == "" ||
		resolvedDesiredVersionFilePath == "" ||
		resolvedCurrentVersionFilePath == resolvedDesiredVersionFilePath {
		return nil
	}

	err = moveToCache(filename, resolvedCurrentVersionFilePath)
	if err != nil {
		return err
	}
	return obtainFile(resolvedDesiredVersionFilePath, multiProgress)
}

func latestFileForVersion(filename, desiredVersion string, versions []string) (string, error) {
	var versionIdx int
	for i, version := range versions {
		if version == desiredVersion {
			versionIdx = i
			break
		}
	}

	prevVersions := versions[:versionIdx+1]
	existsList := make([]bool, len(prevVersions))
	fatalErrors := make(chan error)
	wgDone := make(chan bool)
	wg := sync.WaitGroup{}

	// fmt.Println("Versions to fetch: " + strings.Join(prevVersions, ", "))

	for i, version := range prevVersions {
		wg.Add(1)
		go func(i int, version string) {
			exists, err := remoteFileVersionExists(filename, version)
			if err != nil {
				fatalErrors <- err
				wg.Done()
				return
			}

			existsList[i] = exists
			// fmt.Println("Fetched version " + filename + "." + version + " and got exists: " + strconv.FormatBool(exists))
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

	// fmt.Println("Exists results: ")

	last := len(existsList) - 1
	latestVersion := ""
	for i := range existsList {
		if existsList[last-i] {
			latestVersion = prevVersions[last-i]
			break
		}
	}
	if latestVersion == "" {
		fmt.Println("No valid versions exist for file " + filename + " and desired version " + desiredVersion)
		return "", nil
	}
	versionedFile := filename + "." + latestVersion
	return versionedFile, nil
}

func remoteFileVersionExists(filename, version string) (bool, error) {
	client := getClient()
	path := filename + "." + version
	urlPath := filepath.ToSlash(path)
	resp, err := client.R().Head(urlPath)
	if err != nil {
		return false, err
	}
	if resp.StatusCode() != 200 {
		return false, nil
	}
	return true, nil
}

func obtainFile(filenameWithVersion string, multiProgress *mpb.Progress) error {
	filename := filenameWithVersion[:len(filenameWithVersion)-7] // Remove 7 character version suffix
	outputPath := filepath.Join(config.GamePath, filename)
	cachePath := filepath.Join(config.CachePath, filenameWithVersion)

	// TODO: Probably best to just attempt to rename, and download on failure, instead of doing the Stat beforehand
	info, err := os.Stat(cachePath)
	if os.IsNotExist(err) {
		return downloadRemoteFile(filenameWithVersion, outputPath, multiProgress)
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		fmt.Println("Error: File is dir: " + cachePath)
		return errors.New("Cannot obtain a file that is a directory")
	}
	return moveFileFromCache(cachePath, outputPath)
}

// Download an individual file and place it in the game directory with its original version name
// The files in the game directory should be cached before performing this
func downloadRemoteFile(filenameWithVersion, outputPath string, multiProgress *mpb.Progress) error {
	urlPath := filepath.ToSlash(filenameWithVersion)
	fileName := path.Base(urlPath)
	// fmt.Println("Downloading file " + urlPath + "...")
	fullUrl := fileServerRoot + "/" + urlPath

	var total int64
	bar := multiProgress.AddBar(total,
		mpb.PrependDecorators(
			decor.CountersKibiByte("% .2f / % .2f"),
		),
		mpb.AppendDecorators(
			decor.EwmaETA(decor.ET_STYLE_GO, 90),
			decor.Name(" "+fileName+" "),
			decor.EwmaSpeed(decor.UnitKiB, "% .2f", 60),
		),
		mpb.BarRemoveOnComplete(),
	)

	prevTime := time.Now()
	g := new(got.Got)
	g.ProgressFunc = func(d *got.Download) {
		total = int64(d.TotalSize())
		bar.SetTotal(total, false)
		bar.SetCurrent(int64(d.Size()))
		now := time.Now()
		dur := now.Sub(prevTime)
		bar.DecoratorEwmaUpdate(dur)
		prevTime = now
	}
	err := g.Do(&got.Download{
		URL:  fullUrl,
		Dest: outputPath,
		Header: []got.GotHeader{{
			Key:   "Authorization",
			Value: "Basic " + base64.StdEncoding.EncodeToString([]byte(archiveUser+":"+archivePass)),
		}},
	})
	bar.SetTotal(total, true)

	if err != nil {
		fmt.Println("Unable to download file " + filenameWithVersion)
		fmt.Println(err.Error())
		return err
	}

	// fmt.Println("Finished downloading " + urlPath)
	return nil
}

func getWidth() int {
	if width, _, err := term.GetSize(0); err == nil && width > 0 {
		return width
	}
	return 80
}

// Get a file from the cache and place it in the game directory with its original version name
// The files in the game directory should be cached before performing this
func moveFileFromCache(cachePath, outputPath string) error {
	err := moveFile(cachePath, outputPath)
	if err != nil {
		return err
	}
	return nil
}

func isIgnoredFile(localFilePath string) bool {
	for _, value := range ignoredGameDirs {
		if len(localFilePath) >= len(value) && localFilePath[:len(value)] == value {
			// fmt.Println("Ignoring file in \"" + value + "\" folder")
			return true
		}
	}
	return false
}

func moveToCache(localFilePath, versionFilePath string) error {
	if isIgnoredFile(localFilePath) {
		return nil
	}
	cacheFilePath := filepath.Join(config.CachePath, versionFilePath)
	gameFilePath := filepath.Join(config.GamePath, localFilePath)
	// fmt.Println("Moving file " + gameFilePath + " to " + cacheFilePath)
	dirPath, _ := filepath.Split(cacheFilePath)
	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		return err
	}
	return moveFile(gameFilePath, cacheFilePath)
}

func getLocalPath(filePath string) string {
	pathList := strings.Split(filePath, string(os.PathSeparator))
	gamePathList := strings.Split(config.GamePath, string(os.PathSeparator))
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
	saveFiles := []string{"1.save", "2.save", "3.save", "4.save"}
	for _, saveFile := range saveFiles {
		oldFilePath := filepath.Join(config.SavePath, saveFile)
		newFileName := saveFile + "." + version + "." + strings.Replace(time.Now().Format(time.RFC3339), ":", "-", -1) + ".bak"
		newFilePath := filepath.Join(config.SavePath, newFileName)
		err := os.Rename(oldFilePath, newFilePath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
	}
	return nil
}

func writeDefaultConfig(versions []string) ([]byte, error) {
	gamePath := filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Ubisoft", "Ubisoft Game Launcher", "games", "Watch Dogs Legion")
	cachePath := filepath.Join(gamePath, "..", "Watch Dogs Legion Version Cache")
	savePath, saveErr := getSavePath()
	latestVersion := versions[len(versions)-1]
	cfg := Config{
		CurrentGameVersion: latestVersion,
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
		return nil, errors.New("Unable to automatically detect save file location. Please set manually in in config.yml")
	}
	configFile := filepath.Base(configPath)
	fmt.Println("Wrote the following default values to " + configFile + ":")
	fmt.Println(string(cfgYaml))
	fmt.Println("If you want to change these values, press CTRL+C to exit")
	fmt.Println("To continue, press [ENTER]")
	fmt.Scanln()
	return cfgYaml, nil
}

func getConfig(versions []string) (*Config, error) {
	cfgYaml, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Println("Config file doesn't exist, creating one...")
		cfgYaml, err = writeDefaultConfig(versions)
		if err != nil {
			return nil, err
		}
	}
	cfg := &Config{}
	err = yaml.Unmarshal(cfgYaml, &cfg)
	return cfg, err
}

func setCurrentGameVersion(version string) error {
	config.CurrentGameVersion = version
	cfgYaml, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(configPath, cfgYaml, 0644)
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

func moveFile(sourcePath, destPath string) error {
	err := os.Rename(sourcePath, destPath)
	if terr, ok := err.(*os.LinkError); ok {
		err = handleRenameErr(sourcePath, destPath, terr)
	}
	if err != nil {
		return err
	}
	return nil
}

// handleRenameErr is a helper function that tries to recover from cross-device rename
// errors by falling back to copying.
func handleRenameErr(from, to string, terr *os.LinkError) error {
	// When there are different physical devices we cannot rename cross device.
	// Instead we copy.

	// In windows it can drop down to an operating system call that
	// returns an operating system error with a different number and
	// message. Checking for that as a fall back.
	noerr, ok := terr.Err.(syscall.Errno)
	// 0x11 (ERROR_NOT_SAME_DEVICE) is the windows error.
	// See https://msdn.microsoft.com/en-us/library/cc231199.aspx
	if ok && noerr == 0x11 {
		return copyFileCrossDevice(from, to)
	}
	return terr
}

func copyFileCrossDevice(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return err
	}
	defer outputFile.Close()
	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	if err != nil {
		return err
	}
	// The copy was successful, so now delete the original file
	err = os.Remove(sourcePath)
	if err != nil {
		return err
	}
	return nil
}
