SHELL := /bin/sh

BIN := toon
DIST := dist
PROFILES := profiles
TESTBIN := $(PROFILES)/toon.test
BENCH := ^BenchmarkDecodeSmall$$
CONFORMANCE_ARGS ?=
PPROF ?= $(HOME)/go/bin/pprof

.PHONY: test
test:
	go test ./...

.PHONY: conformance-upstream
conformance-upstream:
	go run ./cmd/toonconformance $(CONFORMANCE_ARGS)

.PHONY: bench
bench:
	go test ./... -run '^$$' -bench . -benchmem

.PHONY: profile-bin
profile-bin:
	mkdir -p $(PROFILES)
	go test -c -o $(TESTBIN) .

.PHONY: profile-cpu
profile-cpu: profile-bin
	$(TESTBIN) -test.run '^$$' -test.bench '$(BENCH)' -test.benchmem -test.cpuprofile $(PROFILES)/cpu.out

.PHONY: profile-mem
profile-mem: profile-bin
	$(TESTBIN) -test.run '^$$' -test.bench '$(BENCH)' -test.benchmem -test.memprofile $(PROFILES)/mem.out

.PHONY: profile-cpu-text
profile-cpu-text: profile-cpu
	$(PPROF) -top -nodecount=200 -cum $(PROFILES)/cpu.out > $(PROFILES)/cpu.txt

.PHONY: profile-mem-text
profile-mem-text: profile-mem
	$(PPROF) -top -nodecount=200 -cum -sample_index=inuse_space $(PROFILES)/mem.out > $(PROFILES)/mem_inuse.txt
	$(PPROF) -top -nodecount=200 -cum -sample_index=alloc_space $(PROFILES)/mem.out > $(PROFILES)/mem_alloc.txt

.PHONY: gc-trace
gc-trace: profile-bin
	GODEBUG=gctrace=1 $(TESTBIN) -test.run '^$$' -test.bench '$(BENCH)' -test.benchmem > $(PROFILES)/gctrace.txt 2>&1

.PHONY: clean
clean:
	rm -rf $(DIST) $(PROFILES)

.PHONY: build
build: clean
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o $(DIST)/$(BIN)_darwin_arm64 ./cmd/toon
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o $(DIST)/$(BIN)_darwin_amd64 ./cmd/toon
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o $(DIST)/$(BIN)_linux_amd64 ./cmd/toon
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o $(DIST)/$(BIN)_linux_arm64 ./cmd/toon
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o $(DIST)/$(BIN)_windows_amd64.exe ./cmd/toon
