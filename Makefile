GOOS=windows
GOARCH=amd64
GOBIN = $(shell go env GOPATH)/bin

default:
	$(GOBIN)/golangci-lint run
	go get github.com/akavel/rsrc
	$(GOBIN)/rsrc -manifest wdl-version-manager.exe.manifest -o wdl-version-manager.syso
	go build -o wdl-version-manager.exe -ldflags="-X internal.archiveUserPack=${ARCHIVE_USER} -X internal.archivePassPack=${ARCHIVE_PASS}" ./cmd/watch-dogs-legion-version-manager
# To run locally: ARCHIVE_USER=username ARCHIVE_PASS=password ./wdl-version-manager.exe