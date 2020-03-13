PREFIX ?= /usr/local

WEBFILES = \
  web/html/index-top.html\
	web/html/index-bottom.html\
	web/html/base.html\
	web/js/app.js\
	web/js/base.js\
	web/js/config.js\
	web/js/configitems.js\
	web/js/controls.js\
	web/js/datasets.js\
	web/js/info.js\
	web/js/init.js\
	web/js/popup.js\
	web/js/state.js\
	web/js/template.js\
	web/css/color.css\
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

all: questscreen/questscreen

web/data.go: ${WEBFILES}
	#go get github.com/go-bindata/go-bindata/...
	${GOPATH}/bin/go-bindata -o web/data.go -pkg web web/favicon web/html web/css web/js web/webfonts

questscreen/questscreen: web/data.go
	cd questscreen && go build

.PHONY: questscreen/questscreen

install: questscreen/questscreen
	cp questscreen/questscreen ${PREFIX}/bin
