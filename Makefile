VERSION ?= "$(shell git describe --tags --long | sed 's/\([^-]*-\)g/r\1/;s/-/./g')"
CGO_CPPFLAGS ?= "$(CPPFLAGS)"
CGO_CFLAGS ?= "$(CFLAGS)"
CGO_CXXFLAGS ?= "$(CXXFLAGS)"
CGO_LDFLAGS ?= "$(LDFLAGS)"
GOFLAGS ?= " -trimpath"

all:
	go build -ldflags \
		"-X main.AppName=url_handler \
		 -X main.VersionNumber=$(VERSION)"

doc:
	scdoc < url_handler.1.scd | gzip - > url_handler.1.gz
	scdoc < url_handler.5.scd | gzip - > url_handler.5.gz
