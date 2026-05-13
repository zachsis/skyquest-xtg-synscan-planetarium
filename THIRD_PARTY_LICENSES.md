# Third-Party Licenses

## OpenNGC

- Project: https://github.com/mattiaverga/OpenNGC
- License: Creative Commons Attribution-ShareAlike 4.0 International (CC BY-SA 4.0)
- Usage: NGC/IC object catalog data embedded in internal/catalog/ngc_filtered.csv

## HYG Star Database

- Project: https://github.com/astronexus/HYG-Database
- License: Creative Commons Attribution-ShareAlike 2.5 Generic (CC BY-SA 2.5)
- Usage: Star position data (RA, Dec, magnitude, spectral type) embedded in internal/catalog/hyg_filtered.csv

## VSOP87 Planetary Theory

- Authors: P. Bretagnon & G. Francou, Bureau des Longitudes, Paris
- Reference: Bretagnon P., Francou G.: 1988, Astron. Astrophys. 202, 309-315
- Source: https://cdsarc.cds.unistra.fr/ftp/cats/VI/81/
- License: Public domain (distributed by CDS Strasbourg)
- Usage: VSOP87B series data files (VSOP87B.mer through VSOP87B.nep) embedded
  in internal/ephemeris/data/ for computing heliocentric ecliptic coordinates
  of the eight planets. Accessed via the github.com/soniakeys/meeus/v3 library.
