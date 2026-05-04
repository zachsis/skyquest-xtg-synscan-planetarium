#!/usr/bin/env python3
"""Filter HYG 4.2 CSV to stars with apparent magnitude <= 7.0."""
import csv
import sys

def main():
    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <hygdata_v42.csv>", file=sys.stderr)
        sys.exit(1)

    with open(sys.argv[1], newline="", encoding="utf-8") as f:
        reader = csv.reader(f)
        header = next(reader)
        mag_idx = header.index("mag")

        writer = csv.writer(sys.stdout)
        writer.writerow(header)
        for row in reader:
            try:
                if float(row[mag_idx]) <= 7.0:
                    writer.writerow(row)
            except (ValueError, IndexError):
                pass

if __name__ == "__main__":
    main()
