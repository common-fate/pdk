PREFIX?=/usr/local

build:
	go build -o ./bin/dpdk cmd/pdk/main.go && mv ./bin/dpdk ${PREFIX}/bin

	