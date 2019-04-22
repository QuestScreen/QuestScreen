all: rpscreen/rpscreen

web/data.go: web/templates/index.html
	go get github.com/go-bindata/go-bindata/...
	${GOPATH}/bin/go-bindata -o web/data.go -pkg web web/templates web/style

rpscreen/rpscreen: web/data.go
	cd rpscreen && go build

.PHONY: rpscreen/rpscreen
