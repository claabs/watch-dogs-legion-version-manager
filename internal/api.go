package internal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-resty/resty/v2"
)

const FileServerRoot = "https://wdlpatches2.charlielaabs.com"

var archiveUserPack string
var archivePassPack string
var archiveUserEnv = os.Getenv("ARCHIVE_USER")
var archivePassEnv = os.Getenv("ARCHIVE_PASS")

var ArchiveUser = CheckEmptyString(archiveUserEnv, archiveUserPack)
var ArchivePass = CheckEmptyString(archivePassEnv, archivePassPack)

var client = GetClient()

func CheckEmptyString(potEmpty, defaultVal string) string {
	if potEmpty == "" {
		return defaultVal
	}
	return potEmpty
}

func GetVersions() ([]string, error) {
	client := GetClient()
	resp, err := client.R().Get("versions.txt")
	if err != nil {
		return nil, err
	}
	statusCode := resp.StatusCode()
	if statusCode != 200 {
		return nil, errors.New("Status code: " + resp.Status())
	}
	unixNewlineBody := strings.ReplaceAll(string(resp.Body()), "\r\n", "\n")
	versions := strings.Split(unixNewlineBody, "\n")
	return versions, nil
}

func GetClient() *resty.Client {
	client := resty.New()
	client.SetBasicAuth(ArchiveUser, ArchivePass)
	client.SetHostURL(FileServerRoot)
	return client
}

func GetFiles() ([]string, error) {
	resp, err := client.R().Get("files.txt")
	if err != nil {
		return nil, err
	}
	statusCode := resp.StatusCode()
	if statusCode != 200 {
		return nil, errors.New("Status code: " + resp.Status())
	}
	unixNewlineBody := strings.ReplaceAll(string(resp.Body()), "\r\n", "\n")
	files := strings.Split(unixNewlineBody, "\n")
	return files, nil
}

func GetSFVLines() ([]string, error) {
	resp, err := client.R().Get("files.sfv")
	if err != nil {
		return nil, err
	}
	statusCode := resp.StatusCode()
	if statusCode != 200 {
		return nil, errors.New("Status code: " + resp.Status())
	}
	unixNewlineBody := strings.ReplaceAll(string(resp.Body()), "\r\n", "\n")
	files := strings.Split(unixNewlineBody, "\n")
	return files, nil
}

func RemoteFileVersionExists(filename, version string) (bool, error) {
	path := filename + "." + version
	urlPath := filepath.ToSlash(path)
	resp, err := client.R().Head(urlPath)
	if err != nil {
		return false, err
	}
	if resp.StatusCode() != 200 {
		return false, nil
	}
	if resp.Header().Get("Content-Length") == "0" {
		return false, nil
	}
	return true, nil
}

func DownloadRemoteFileSlow(filenameWithVersion, outputPath string) error {
	urlPath := filepath.ToSlash(filenameWithVersion)
	fmt.Println("Downloading file: " + urlPath)
	_, err := client.R().SetOutput(outputPath).Get(urlPath)
	if err != nil {
		return err
	}
	fmt.Println("Finished downloading: " + urlPath)
	return nil
}
