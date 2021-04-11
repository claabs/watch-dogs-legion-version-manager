package internal

import (
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"time"

	"github.com/machinebox/progress"
	"github.com/vbauerster/mpb/v6"
	"github.com/vbauerster/mpb/v6/decor"
)

func HashFileCRC32(filePath string, multiProgress *mpb.Progress) (uint32, error) {
	var returnCRC32UInt uint32
	fileStat, err := os.Stat(filePath)
	if err != nil {
		return returnCRC32UInt, err
	}
	fileSize := fileStat.Size()

	file, err := os.Open(filePath)
	if err != nil {
		return returnCRC32UInt, err
	}
	defer file.Close()
	var fileReader io.Reader = file // os.File is an io.Reader
	progressReader := progress.NewReader(fileReader)

	bar := multiProgress.AddBar(fileSize,
		mpb.PrependDecorators(
			decor.CountersKibiByte("% .2f / % .2f"),
		),
		mpb.AppendDecorators(
			decor.EwmaETA(decor.ET_STYLE_GO, 90),
			decor.Name(" verifying "+filePath+" "),
			decor.EwmaSpeed(decor.UnitKiB, "% .2f", 60),
		),
		mpb.BarRemoveOnComplete(),
	)

	prevTime := time.Now()
	ctx := context.Background()
	go func() {
		progressChan := progress.NewTicker(ctx, progressReader, fileSize, 1*time.Second)
		for p := range progressChan {
			bar.SetTotal(fileSize, false)
			bar.SetCurrent(p.N())
			now := time.Now()
			dur := now.Sub(prevTime)
			bar.DecoratorEwmaUpdate(dur)
			prevTime = now
		}
		bar.SetTotal(fileSize, true)
		fmt.Println("\r hash is completed")
	}()

	tablePolynomial := crc32.IEEETable
	hash := crc32.New(tablePolynomial)
	if _, err := io.Copy(hash, file); err != nil {
		return returnCRC32UInt, err
	}
	returnCRC32UInt = hash.Sum32()
	return returnCRC32UInt, nil

}
