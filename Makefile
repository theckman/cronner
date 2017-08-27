dep:
	go get -u github.com/golang/dep/cmd/dep

test:
	go test -v -check.vv ./...

build: test
	go build -o cronner .

prep-deploy:
	mkdir build
	mkdir ${PROJECT_BUILD_NAME}
	go build -o ${PROJECT_BUILD_NAME}/${PROJECT_NAME}
	tar -czf build/${PROJECT_BUILD_NAME}.tar.gz ${PROJECT_BUILD_NAME}/
	shasum -a 256 -- build/${PROJECT_BUILD_NAME}.tar.gz | sed -e 's#build/##g' > build/${PROJECT_BUILD_NAME}.tar.gz.sha256
	rm -rf ${PROJECT_BUILD_NAME}

.PHONY: dep test build prep-deploy
