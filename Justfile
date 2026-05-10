default:
    @just --list

build:
    go build ./...

test:
    @if find cmd internal -name '*.go' 2>/dev/null | head -1 | grep -q .; then \
        go test -race -cover ./...; \
    else \
        echo "no Go source yet — test skipped"; \
    fi

lint:
    @if find cmd internal -name '*.go' 2>/dev/null | head -1 | grep -q .; then \
        golangci-lint run; \
    else \
        echo "no Go source yet — lint skipped"; \
    fi

fmt:
    gofumpt -w .
    goimports -w .

run *args:
    go run ./cmd/builder {{args}}
