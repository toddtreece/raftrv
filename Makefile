GO_FILES := $(shell find . -name '*.go')

raftrv: $(GO_FILES) clean
	go build .

.PHONY: clean
clean:
	rm -rf raftexample-*

.PHONY: run
run: clean raftrv
	goreman start

.PHONY: test
test:
	k6 run k6.js