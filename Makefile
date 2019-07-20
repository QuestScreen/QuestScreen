all: rpscreen/rpscreen

web/data.go: web/templates/index.html web/style/style.css web/style/pure-min.css web/js/ui.js
	go get github.com/go-bindata/go-bindata/...
	${GOPATH}/bin/go-bindata -o web/data.go -pkg web web/templates web/style web/js

rpscreen/rpscreen: web/data.go
	cd rpscreen && go build

.PHONY: rpscreen/rpscreen
