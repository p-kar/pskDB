all: server client
	go build -o master

server:	./server/*
	go build -o serverNode ./server

client: ./client/*
	go build -o clientNode ./client

clean:
	rm -rf master
	rm -rf serverNode
	rm -rf clientNode

cleanup:
	-pkill clientNode
	-pkill serverNode