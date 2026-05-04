.PHONY: build run test lint clean generate-catalog

build:
	go build -o bin/oriontelescope ./cmd/oriontelescope

run:
	go run ./cmd/oriontelescope

test:
	go test ./...

lint:
	go vet ./...
	staticcheck ./...

clean:
	rm -rf bin/

generate-catalog:
	@echo "Downloading full HYG 4.2 CSV..."
	curl -sL -o /tmp/hygdata_v42.csv \
	    "https://raw.githubusercontent.com/astronexus/HYG-Database/refs/heads/main/hyg/CURRENT/hygdata_v41.csv"
	@echo "Filtering to mag <= 7.0..."
	python3 scripts/filter_hyg.py /tmp/hygdata_v42.csv > internal/catalog/hyg_filtered.csv
	@echo "Done: internal/catalog/hyg_filtered.csv"
