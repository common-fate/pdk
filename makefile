PREFIX?=/usr/local

cli:
	go build -o ./bin/pdk cmd/main.go && mv ./bin/pdk ${PREFIX}/bin
