PREFIX?=/usr/local

build:
	go build -o ./bin/dpdk cmd/main.go && mv ./bin/dpdk ${PREFIX}/bin

	