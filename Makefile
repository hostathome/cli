.PHONY: build install clean test release deb

BINARY=hostathome
VERSION?=0.1.0
LDFLAGS=-ldflags "-X main.cliVersion=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/hostathome

install: build
	cp bin/$(BINARY) $(GOPATH)/bin/$(BINARY) 2>/dev/null || \
	sudo cp bin/$(BINARY) /usr/local/bin/$(BINARY)

clean:
	rm -rf bin/ dist/

test:
	go test ./...

# Build for all platforms
release: clean
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-linux-amd64 ./cmd/hostathome
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY)-linux-arm64 ./cmd/hostathome
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-darwin-amd64 ./cmd/hostathome
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY)-darwin-arm64 ./cmd/hostathome
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-windows-amd64.exe ./cmd/hostathome

# Build .deb package for Debian/Ubuntu
deb:
	@echo "Building .deb package..."
	mkdir -p dist/deb/$(BINARY)_$(VERSION)_amd64/DEBIAN
	mkdir -p dist/deb/$(BINARY)_$(VERSION)_amd64/usr/bin
	mkdir -p dist/deb/$(BINARY)_$(VERSION)_amd64/usr/share/doc/$(BINARY)

	# Build binary
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/deb/$(BINARY)_$(VERSION)_amd64/usr/bin/$(BINARY) ./cmd/hostathome

	# Create control file
	@echo "Package: $(BINARY)" > dist/deb/$(BINARY)_$(VERSION)_amd64/DEBIAN/control
	@echo "Version: $(VERSION)" >> dist/deb/$(BINARY)_$(VERSION)_amd64/DEBIAN/control
	@echo "Section: games" >> dist/deb/$(BINARY)_$(VERSION)_amd64/DEBIAN/control
	@echo "Priority: optional" >> dist/deb/$(BINARY)_$(VERSION)_amd64/DEBIAN/control
	@echo "Architecture: amd64" >> dist/deb/$(BINARY)_$(VERSION)_amd64/DEBIAN/control
	@echo "Depends: docker.io | docker-ce" >> dist/deb/$(BINARY)_$(VERSION)_amd64/DEBIAN/control
	@echo "Maintainer: HostAtHome <hello@hostathome.dev>" >> dist/deb/$(BINARY)_$(VERSION)_amd64/DEBIAN/control
	@echo "Description: Manage game servers with Docker" >> dist/deb/$(BINARY)_$(VERSION)_amd64/DEBIAN/control
	@echo " HostAtHome CLI lets you install, run, and manage" >> dist/deb/$(BINARY)_$(VERSION)_amd64/DEBIAN/control
	@echo " game servers using Docker containers." >> dist/deb/$(BINARY)_$(VERSION)_amd64/DEBIAN/control
	@echo " Supported games: Minecraft, Project Zomboid, CS2" >> dist/deb/$(BINARY)_$(VERSION)_amd64/DEBIAN/control

	# Create copyright file
	@echo "Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/" > dist/deb/$(BINARY)_$(VERSION)_amd64/usr/share/doc/$(BINARY)/copyright
	@echo "Upstream-Name: $(BINARY)" >> dist/deb/$(BINARY)_$(VERSION)_amd64/usr/share/doc/$(BINARY)/copyright
	@echo "Source: https://github.com/hostathome/cli" >> dist/deb/$(BINARY)_$(VERSION)_amd64/usr/share/doc/$(BINARY)/copyright
	@echo "" >> dist/deb/$(BINARY)_$(VERSION)_amd64/usr/share/doc/$(BINARY)/copyright
	@echo "Files: *" >> dist/deb/$(BINARY)_$(VERSION)_amd64/usr/share/doc/$(BINARY)/copyright
	@echo "Copyright: 2024 HostAtHome" >> dist/deb/$(BINARY)_$(VERSION)_amd64/usr/share/doc/$(BINARY)/copyright
	@echo "License: MIT" >> dist/deb/$(BINARY)_$(VERSION)_amd64/usr/share/doc/$(BINARY)/copyright

	# Build package
	dpkg-deb --build dist/deb/$(BINARY)_$(VERSION)_amd64
	mv dist/deb/$(BINARY)_$(VERSION)_amd64.deb dist/
	rm -rf dist/deb/$(BINARY)_$(VERSION)_amd64

	# Create version-less copy for /latest/ downloads
	cp dist/$(BINARY)_$(VERSION)_amd64.deb dist/$(BINARY)_amd64.deb

	@echo "Created: dist/$(BINARY)_$(VERSION)_amd64.deb"
	@echo "Created: dist/$(BINARY)_amd64.deb (for /latest/)"
	@echo ""
	@echo "Install with: sudo dpkg -i dist/$(BINARY)_$(VERSION)_amd64.deb"

# Build .deb for arm64
deb-arm64:
	@echo "Building .deb package for arm64..."
	mkdir -p dist/deb/$(BINARY)_$(VERSION)_arm64/DEBIAN
	mkdir -p dist/deb/$(BINARY)_$(VERSION)_arm64/usr/bin
	mkdir -p dist/deb/$(BINARY)_$(VERSION)_arm64/usr/share/doc/$(BINARY)

	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/deb/$(BINARY)_$(VERSION)_arm64/usr/bin/$(BINARY) ./cmd/hostathome

	@echo "Package: $(BINARY)" > dist/deb/$(BINARY)_$(VERSION)_arm64/DEBIAN/control
	@echo "Version: $(VERSION)" >> dist/deb/$(BINARY)_$(VERSION)_arm64/DEBIAN/control
	@echo "Section: games" >> dist/deb/$(BINARY)_$(VERSION)_arm64/DEBIAN/control
	@echo "Priority: optional" >> dist/deb/$(BINARY)_$(VERSION)_arm64/DEBIAN/control
	@echo "Architecture: arm64" >> dist/deb/$(BINARY)_$(VERSION)_arm64/DEBIAN/control
	@echo "Depends: docker.io | docker-ce" >> dist/deb/$(BINARY)_$(VERSION)_arm64/DEBIAN/control
	@echo "Maintainer: HostAtHome <hello@hostathome.dev>" >> dist/deb/$(BINARY)_$(VERSION)_arm64/DEBIAN/control
	@echo "Description: Manage game servers with Docker" >> dist/deb/$(BINARY)_$(VERSION)_arm64/DEBIAN/control
	@echo " HostAtHome CLI lets you install, run, and manage" >> dist/deb/$(BINARY)_$(VERSION)_arm64/DEBIAN/control
	@echo " game servers using Docker containers." >> dist/deb/$(BINARY)_$(VERSION)_arm64/DEBIAN/control
	@echo " Supported games: Minecraft, Project Zomboid, CS2" >> dist/deb/$(BINARY)_$(VERSION)_arm64/DEBIAN/control

	@echo "Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/" > dist/deb/$(BINARY)_$(VERSION)_arm64/usr/share/doc/$(BINARY)/copyright
	@echo "Upstream-Name: $(BINARY)" >> dist/deb/$(BINARY)_$(VERSION)_arm64/usr/share/doc/$(BINARY)/copyright
	@echo "Source: https://github.com/hostathome/cli" >> dist/deb/$(BINARY)_$(VERSION)_arm64/usr/share/doc/$(BINARY)/copyright

	dpkg-deb --build dist/deb/$(BINARY)_$(VERSION)_arm64
	mv dist/deb/$(BINARY)_$(VERSION)_arm64.deb dist/
	rm -rf dist/deb/$(BINARY)_$(VERSION)_arm64

	# Create version-less copy for /latest/ downloads
	cp dist/$(BINARY)_$(VERSION)_arm64.deb dist/$(BINARY)_arm64.deb

	@echo "Created: dist/$(BINARY)_$(VERSION)_arm64.deb"
	@echo "Created: dist/$(BINARY)_arm64.deb (for /latest/)"
