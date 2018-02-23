GOPATH=$(shell pwd)
export GOPATH
COLORDIR=pkg/

SERVERDIR=server/
CLIENTDIR=client/
COMMONDIR=common/

SERVERSRC=$(wildcard $(SERVERDIR)/*.go)
CLIENTSRC=$(wildcard $(CLIENTDIR)/*.go)
COMMONSRC=$(wildcard $(COMMONDIR)/*.go)

all: color master serverNode clientNode

	
color: 
	@if [ ! -d $(COLORDIR) ]; then \
		go get github.com/fatih/color; \
	fi

master: master.go $(COMMONSRC)
	go build -o master
	
serverNode: $(SERVERSRC) $(COMMONSRC)
	go build -o serverNode ./server

clientNode: $(CLIENTSRC) $(COMMONSRC)
	go build -o clientNode ./client

clean:
	rm -rf master
	rm -rf serverNode
	rm -rf clientNode

cleanup:
	@echo -n 'Killing stray server and client processes... '
	@-pkill serverNode clientNode | true
	@echo 'done.'
