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

var checksums map[string]uint32

func PopulateChecksums() error {
	lines, err := GetSFVLines()
	if err != nil {
		return err
	}
	checksums, err = ParseChecksums(lines)
	return err
}

func ParseChecksums(lines []string) (map[string]uint32, error) {
	checksumAcc := make(map[string]uint32)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, ";") {
			continue
		}
		checksum, err := ParseChecksum(line)
		if err != nil {
			return nil, err
		}
		checksumAcc[checksum.Filename] = checksum.CRC32
	}
	return checksumAcc, nil
}

func ParseChecksum(line string) (*Checksum, error) {
	parts := strings.SplitN(line, " ", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("could not parse checksum: %q", line)
	}
	filename := strings.TrimSpace(parts[0])
	crc32, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 16, 32)
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
	if len(checksums) == 0 {
		err := PopulateChecksums()
		if err != nil {
			return 0, err
		}
	}
	crc := checksums[filename]
	if crc == 0 {
		return 0, errors.New("Could not find CRC32 for file: " + filename)
	}
	return crc, nil
}
