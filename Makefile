GOOS=windows
GOARCH=amd64

default:
	golangci-lint run
	go get github.com/akavel/rsrc
	rsrc -manifest wdl-version-manager.exe.manifest -o wdl-version-manager.syso
	go build -o wdl-version-manager.exe ./cmd/watch-dogs-legion-version-manager
# To run locally: ARCHIVE_USER=username ARCHIVE_PASS=password ./wdl-version-manager.exe