build: 
	@go build -o bin/redpoker 

run: build 
	@./bin/redpoker 

test:
	go test -v ./...