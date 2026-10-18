.PHONY: fmt vet test check

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './.git/*')

vet:
	go vet ./...

test:
	go test ./...

check: vet test
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt 未格式化:"; echo "$$out"; exit 1; fi
