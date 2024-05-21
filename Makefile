
GOFILES := $(wildcard **/*.go)

.PHONY: test
test: cov.out

cov.out: $(GOFILES)
	go test -coverprofile=cov.out ./...

.PHONY: htmlcov
htmlcov: cov.out
	go tool cover -html cov.out
