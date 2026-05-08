.PHONY: build run test lint clean generate-catalog generate-ngc

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

generate-ngc:
	@echo "Downloading OpenNGC CSV..."
	curl -sL -o /tmp/NGC_raw.csv \
	    "https://raw.githubusercontent.com/mattiaverga/OpenNGC/master/database_files/NGC.csv"
	@echo "Filtering to objects with known coordinates..."
	python3 scripts/filter_ngc.py /tmp/NGC_raw.csv internal/catalog/ngc_filtered.csv
	@echo "Done: internal/catalog/ngc_filtered.csv"
