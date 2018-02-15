all: server client
	go build -o master
	go build -o serverNode ./server
	go build -o clientNode ./client

clean:
	rm -rf master
	rm -rf serverNode
	rm -rf clientNode

cleanup:
	-pkill clientNode
	-pkill serverNode
