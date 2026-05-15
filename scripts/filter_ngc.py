#!/usr/bin/env python3
"""Filter the OpenNGC CSV to objects with known coordinates and magnitude.

Usage: python3 scripts/filter_ngc.py <input_csv> <output_csv>

Input: OpenNGC NGC.csv from https://github.com/mattiaverga/OpenNGC
Output: Filtered CSV with columns: Name, Type, RA, Dec, Mag, MajAx, Constellation, CommonName
"""
import csv
import sys
import math


def hms_to_hours(hms: str) -> float:
    """Convert HH:MM:SS.ss to decimal hours."""
    parts = hms.strip().split(":")
    if len(parts) != 3:
        raise ValueError(f"invalid HMS: {hms}")
    h = float(parts[0])
    m = float(parts[1])
    s = float(parts[2])
    return h + m / 60 + s / 3600


def dms_to_deg(dms: str) -> float:
    """Convert +DD:MM:SS.s to decimal degrees."""
    dms = dms.strip()
    sign = -1 if dms.startswith("-") else 1
    dms = dms.lstrip("+-")
    parts = dms.split(":")
    if len(parts) != 3:
        raise ValueError(f"invalid DMS: {dms}")
    d = float(parts[0])
    m = float(parts[1])
    s = float(parts[2])
    return sign * (d + m / 60 + s / 3600)


def main():
    if len(sys.argv) != 3:
        print(__doc__)
        sys.exit(1)

    input_path, output_path = sys.argv[1], sys.argv[2]

    with open(input_path, "r", encoding="utf-8") as f:
        reader = csv.DictReader(f, delimiter=";")
        rows = list(reader)

    out_fields = ["Name", "Type", "RA", "Dec", "Mag", "MajAx", "Constellation", "CommonName"]
    written = 0

    with open(output_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=out_fields)
        writer.writeheader()

        for row in rows:
            name = row.get("Name", "").strip()
            ra_str = row.get("RA", "").strip()
            dec_str = row.get("Dec", "").strip()
            mag_str = row.get("V-Mag", "").strip()
            obj_type = row.get("Type", "").strip()
            constellation = row.get("Const", "").strip()
            major_ax = row.get("MajAx", "").strip()
            common = row.get("Common names", "").strip()

            # Skip objects without coordinates.
            if not ra_str or not dec_str:
                continue

            # Skip objects without magnitude (unless they are nebulae which often lack it).
            if not mag_str and obj_type not in ("HII", "Neb", "RfN", "EmN", "SNR", "PN"):
                continue

            try:
                ra_hours = hms_to_hours(ra_str)
                dec_deg = dms_to_deg(dec_str)
            except (ValueError, IndexError):
                continue

            mag = mag_str if mag_str else ""
            maj_ax = major_ax if major_ax else ""

            # Take first common name if multiple.
            if "," in common:
                common = common.split(",")[0].strip()

            writer.writerow({
                "Name": name,
                "Type": obj_type,
                "RA": f"{ra_hours:.6f}",
                "Dec": f"{dec_deg:.6f}",
                "Mag": mag,
                "MajAx": maj_ax,
                "Constellation": constellation,
                "CommonName": common,
            })
            written += 1

    print(f"Wrote {written} objects to {output_path}")


if __name__ == "__main__":
    main()
