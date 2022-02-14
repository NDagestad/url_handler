# For when I add tags
#VERSION="$(shell git describe --long | sed 's/\([^-]*-\)g/r\1/;s/-/./g')"
VERSION="r$(shell git rev-list --count HEAD).$(shell git rev-parse --short HEAD)"

all:
	go build -ldflags \
		"-X main.AppName=url_handler \
		 -X main.VersionNumber=$(VERSION)"

doc:
	scdoc < url_handler.1.scd > url_handler.1
	scdoc < url_handler.5.scd > url_handler.5
