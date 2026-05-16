default:
    @just --list

build:
    go build -o builder ./cmd/builder

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

# Run integration tests (require Node.js >= 18 + @angular-devkit/schematics-cli >= 17 on PATH).
# Skips gracefully if Node or schematics-cli is not available.
test-integration:
    go test -v -tags=integration -timeout 60s ./internal/shared/engine/angular/...

# Run all fitness functions against the real codebase. Must exit 0.
# Enforces: fitness-functions-ci.REQ-01.1 through .09 + builder-init-end-to-end FF-init-01..04
# + cli-versioning-automation FF-14, FF-21, FF-22 (S-0/S-1); FF-16..FF-20, FF-23 (S-2)
# + color-palette-theming FF-24 = 23 total
fitness:
    @bash scripts/fitness/handler-loc.sh
    @bash scripts/fitness/no-cross-feature.sh
    @bash scripts/fitness/no-concrete-adapter-from-features.sh
    @bash scripts/fitness/shared-isolation.sh
    @bash scripts/fitness/iface-asserts.sh
    @bash scripts/fitness/cobra-help-listing.sh
    @bash scripts/fitness/input-reply-ctx-guard.sh
    @bash scripts/fitness/no-untyped-args.sh
    @bash scripts/fitness/no-percent-v.sh
    @bash scripts/fitness/no-direct-os-io.sh
    @bash scripts/fitness/init-marker-uniqueness.sh
    @bash scripts/fitness/init-errcode-additive.sh
    @bash scripts/fitness/init-skill-bytes-stable.sh
    @bash scripts/fitness/version-const-regex.sh
    @bash scripts/fitness/codeowners-workflows.sh
    @bash scripts/fitness/pr-bump-label-validator.sh
    @bash scripts/fitness/release-workflow-permissions.sh
    @bash scripts/fitness/release-workflow-concurrency.sh
    @bash scripts/fitness/release-workflow-no-pat.sh
    @bash scripts/fitness/workflows-sha-pinned.sh
    @bash scripts/fitness/release-anti-loop-guard.sh
    @bash scripts/fitness/bump-version-script.sh
    @bash scripts/fitness/no-hex-leak.sh

# FF-24: Check for raw hex color literals outside internal/shared/render/theme/.
# Enforces theme-tokens/REQ-03.1, render-pretty/REQ-05.1, REQ-05.2.
fitness-hex-leak:
    @bash scripts/fitness/no-hex-leak.sh

# Run the bump-version.sh test driver (8+ cases covering arithmetic + edge cases).
# Enforces: cli-versioning-automation REQ-CVA-040, REQ-CVA-041
test-bump:
    @sh scripts/release/bump-version.test.sh

# Run each fitness function against its bad-pattern fixture.
# Each invocation MUST exit non-zero (the fixture triggers the violation).
# The meta-target inverts each script's exit code: success = rule caught violation.
# Enforces: fitness-functions-ci.REQ-01.1, .02.1, .07.1, .08, .09 (meta-fixture coverage)
fitness-meta:
    @bash scripts/fitness/_meta_invert.sh scripts/fitness/handler-loc.sh testdata/fitness/big-handler.go.txt
    @bash scripts/fitness/_meta_invert.sh scripts/fitness/no-cross-feature.sh testdata/fitness/cross-feature.go.txt
    @bash scripts/fitness/_meta_invert.sh scripts/fitness/input-reply-ctx-guard.sh testdata/fitness/unguarded-send.go.txt
    @bash scripts/fitness/_meta_invert.sh scripts/fitness/no-untyped-args.sh testdata/fitness/untyped-args.go.txt
    @bash scripts/fitness/_meta_invert.sh scripts/fitness/no-percent-v.sh testdata/fitness/percent-v-message.go.txt
