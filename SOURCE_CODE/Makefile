.PHONY: all build install clean

PREFIX ?= /usr/local
BINDIR := $(PREFIX)/bin
ETCDIR := /etc/nossh

all: build

build:
	go build -o bin/nossh-server ./cmd/nossh-server
	go build -o bin/nossh-agent ./cmd/nossh-agent
	go build -o bin/nossh ./cmd/nossh

gui-deb:
	bash scripts/build-gui-deb.sh

windows:
	bash scripts/pack-windows.sh

install: build
	install -d $(BINDIR) $(ETCDIR)
	install -m 755 bin/nossh-server bin/nossh-agent bin/nossh $(BINDIR)/
	install -d /var/lib/nossh
	@if [ ! -f $(ETCDIR)/server.yaml ]; then \
		install -m 600 deploy/server.yaml.example $(ETCDIR)/server.yaml; \
		echo "Edit $(ETCDIR)/server.yaml and set install_token + admin_token"; \
	fi
	install -m 644 deploy/systemd/nossh-server.service /etc/systemd/system/
	systemctl daemon-reload

clean:
	rm -rf bin/
