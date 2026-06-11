#!/usr/bin/env bash
#
# build-contour-tiles.sh — generate self-hosted contour vector tiles for
# the outdoor map style (components/maps/topoStyle.ts).
#
# Everything is free: SRTM-derived 30 m elevation data from the AWS Open
# Data "Terrain Tiles" bucket, processed entirely with GDAL (>= 3.x).
# No tippecanoe, no API keys.
#
# Output: a static {z}/{x}/{y}.pbf tree (layer "contour", integer field
# "ele" in metres). Host it on any static file server / CDN, then set
#
#   EXPO_PUBLIC_CONTOUR_TILES_URL=https://your-host/contours/{z}/{x}/{y}.pbf
#
# and rebuild the app. Without the env var the style simply omits contours.
#
# Usage (defaults cover the Kashmir valley trekking region):
#   ./scripts/build-contour-tiles.sh
#   LAT_MIN=32 LAT_MAX=36 LON_MIN=73 LON_MAX=77 ./scripts/build-contour-tiles.sh
#
# Tunables (env): LAT_MIN/LAT_MAX/LON_MIN/LON_MAX  whole-degree bbox
#                 INTERVAL  contour interval in metres   (default 100)
#                 MINZOOM/MAXZOOM  tile zoom range        (default 9–13)
#                 OUT       output directory              (default build/contour-tiles)

set -euo pipefail

LAT_MIN="${LAT_MIN:-33}"
LAT_MAX="${LAT_MAX:-35}"   # exclusive — DEM tiles N33..N34 cover lat 33–35
LON_MIN="${LON_MIN:-74}"
LON_MAX="${LON_MAX:-76}"   # exclusive
INTERVAL="${INTERVAL:-100}"
MINZOOM="${MINZOOM:-9}"
MAXZOOM="${MAXZOOM:-13}"
OUT="${OUT:-build/contour-tiles}"

for tool in curl gdalbuildvrt gdal_contour ogr2ogr; do
  command -v "$tool" >/dev/null || { echo "error: $tool not found (brew install gdal)"; exit 1; }
done

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

echo "→ downloading DEM tiles (lat ${LAT_MIN}–${LAT_MAX}, lon ${LON_MIN}–${LON_MAX})…"
for lat in $(seq "$LAT_MIN" $((LAT_MAX - 1))); do
  for lon in $(seq "$LON_MIN" $((LON_MAX - 1))); do
    name="$(printf 'N%02dE%03d' "$lat" "$lon")"
    url="https://s3.amazonaws.com/elevation-tiles-prod/skadi/N${lat}/${name}.hgt.gz"
    echo "   ${name}"
    curl -sf -o "$WORK/${name}.hgt.gz" "$url" || { echo "error: failed to fetch $url"; exit 1; }
    gunzip -f "$WORK/${name}.hgt.gz"
  done
done

echo "→ building virtual mosaic…"
gdalbuildvrt -q "$WORK/dem.vrt" "$WORK"/*.hgt

echo "→ tracing ${INTERVAL} m contours (this is the slow part)…"
gdal_contour -q -i "$INTERVAL" -a ele -nln contour "$WORK/dem.vrt" "$WORK/contours.gpkg"

echo "→ encoding vector tiles z${MINZOOM}–z${MAXZOOM}…"
rm -rf "$OUT"
mkdir -p "$(dirname "$OUT")"
ogr2ogr -f MVT "$OUT" "$WORK/contours.gpkg" \
  -dsco MINZOOM="$MINZOOM" -dsco MAXZOOM="$MAXZOOM" -dsco COMPRESS=NO

count="$(find "$OUT" -name '*.pbf' | wc -l | tr -d ' ')"
size="$(du -sh "$OUT" | cut -f1)"
echo
echo "✓ done — ${count} tiles, ${size} in ${OUT}"
echo
echo "Next steps:"
echo "  1. Upload ${OUT}/ to any static host (Cloudflare R2/Pages, S3, your API)."
echo "  2. Set EXPO_PUBLIC_CONTOUR_TILES_URL=https://<host>/<path>/{z}/{x}/{y}.pbf in .env"
echo "  3. Rebuild the app — the outdoor style picks contours up automatically."
