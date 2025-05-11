.PHONY: clean build deploy

build:
	env GOARCH=arm64 GOOS=linux go build -ldflags="-s -w" -o bin/bootstrap .
	chmod +x bin/bootstrap
	zip -j bin/api.zip bin/bootstrap

clean:
	rm -rf ./bin

deploy: clean build
	sls deploy --stage prd --verbose
