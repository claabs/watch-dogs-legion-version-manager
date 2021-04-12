package internal

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Inspiration: https://github.com/cwlbraa/verifysfv/blob/master/sfv/sfv.go

type Checksum struct {
	Filename string
	CRC32    uint32
}

var checksumsMap map[string]uint32
var filenamesMap map[uint32]string

func PopulateChecksums() error {
	lines, err := GetSFVLines()
	if err != nil {
		return err
	}
	checksumsMap, filenamesMap, err = ParseChecksums(lines)
	return err
}

func ParseChecksums(lines []string) (map[string]uint32, map[uint32]string, error) {
	checksumAcc := make(map[string]uint32)
	filenameAcc := make(map[uint32]string)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, ";") {
			continue
		}
		checksum, err := ParseChecksum(line)
		if err != nil {
			return nil, nil, err
		}
		checksumAcc[checksum.Filename] = checksum.CRC32
		filenameAcc[checksum.CRC32] = checksum.Filename
	}
	return checksumAcc, filenameAcc, nil
}

func ParseChecksum(line string) (*Checksum, error) {
	parts := strings.SplitN(line, " ", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("could not parse checksum: %q", line)
	}
	filename := strings.TrimSpace(parts[0])
	crc32, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 32)
	if err != nil {
		return nil, err
	}
	// ParseUint will return error if number exceeds 32 bits
	return &Checksum{
		Filename: filename,
		CRC32:    uint32(crc32),
	}, nil
}

func GetCRC32(filename string) (uint32, error) {
	if len(checksumsMap) == 0 {
		err := PopulateChecksums()
		if err != nil {
			return 0, err
		}
	}
	crc := checksumsMap[filename]
	if crc == 0 {
		return 0, errors.New("Could not find CRC32 for file: " + filename)
	}
	return crc, nil
}

func GetFilename(crc32 uint32) (string, error) {
	if len(checksumsMap) == 0 {
		err := PopulateChecksums()
		if err != nil {
			return "", err
		}
	}
	filename := filenamesMap[crc32]
	if filename == "" {
		return "", errors.New("Could not find filename for crc32: " + strconv.FormatUint(uint64(crc32), 10))
	}
	return filename, nil
}
