
web/data.go:
	go get github.com/go-bindata/go-bindata/...
	$GOPATH/bin/go-bindata -o web/data.go -pkg web web/templates

rpscreen: web/data.go
	go build rpscreen

.PHONY: rpscreen