PREFIX ?= /usr/local

WEBFILES = \
  web/templates/index.html\
	web/js/ui.js\
	web/js/sharedData.js\
	web/css/style.css\
  web/css/pure-min.css\
	web/css/grids-responsive-min.css\
	web/css/fontawesome.min.css\
	web/css/solid.min.css\
	web/webfonts/fa-solid-900.eot\
	web/webfonts/fa-solid-900.svg\
	web/webfonts/fa-solid-900.ttf\
	web/webfonts/fa-solid-900.woff\
	web/webfonts/fa-solid-900.woff2

all: pnpscreen/pnpscreen

web/data.go: ${WEBFILES}
	go get github.com/go-bindata/go-bindata/...
	${GOPATH}/bin/go-bindata -o web/data.go -pkg web web/templates web/css web/js web/webfonts

pnpscreen/pnpscreen: web/data.go
	cd pnpscreen && go build

.PHONY: pnpscreen/pnpscreen

install: pnpscreen/pnpscreen
	cp pnpscreen/pnpscreen ${PREFIX}/bin