#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 OUTPUT_ICNS" >&2
  exit 1
fi

OUTPUT_ICNS="$1"
TMP_DIR="$(mktemp -d)"
ICONSET_DIR="$TMP_DIR/TokenHeat.iconset"
SOURCE_PNG="$TMP_DIR/source.png"
mkdir -p "$ICONSET_DIR"

xcrun swift - "$SOURCE_PNG" <<'SWIFT'
import AppKit

let outputPath = CommandLine.arguments[1]
let size: CGFloat = 1024

let image = NSImage(size: NSSize(width: size, height: size))
image.lockFocus()

let bounds = NSRect(x: 0, y: 0, width: size, height: size)
let background = NSBezierPath(roundedRect: bounds, xRadius: size * 0.22, yRadius: size * 0.22)
NSColor(calibratedWhite: 0.96, alpha: 1).setFill()
background.fill()

let inset = size * 0.12
let plotRect = bounds.insetBy(dx: inset, dy: inset)

let axis = NSBezierPath()
axis.lineWidth = size * 0.016
axis.lineCapStyle = .round
NSColor(calibratedWhite: 0.78, alpha: 1).setStroke()
axis.move(to: CGPoint(x: plotRect.minX, y: plotRect.minY))
axis.line(to: CGPoint(x: plotRect.minX, y: plotRect.maxY))
axis.move(to: CGPoint(x: plotRect.minX, y: plotRect.minY))
axis.line(to: CGPoint(x: plotRect.maxX, y: plotRect.minY))
axis.stroke()

let bars: [(CGFloat, CGFloat)] = [
    (0.12, 0.24),
    (0.30, 0.46),
    (0.48, 0.34),
    (0.66, 0.70),
    (0.84, 0.56),
]
let barWidth = plotRect.width * 0.10

for (index, bar) in bars.enumerated() {
    let x = plotRect.minX + plotRect.width * bar.0 - barWidth / 2
    let height = plotRect.height * bar.1
    let rect = NSRect(x: x, y: plotRect.minY, width: barWidth, height: height)
    let path = NSBezierPath(roundedRect: rect, xRadius: barWidth / 2, yRadius: barWidth / 2)
    let alpha = 0.34 + CGFloat(index) * 0.13
    NSColor(calibratedRed: 0.15, green: 0.42, blue: 0.96, alpha: alpha).setFill()
    path.fill()
}

let dotSize = barWidth
let dotRect = NSRect(
    x: plotRect.maxX - dotSize * 1.1,
    y: plotRect.minY + plotRect.height * 0.74,
    width: dotSize,
    height: dotSize
)
NSColor(calibratedRed: 0.15, green: 0.42, blue: 0.96, alpha: 1).setFill()
NSBezierPath(ovalIn: dotRect).fill()

image.unlockFocus()

guard
    let tiff = image.tiffRepresentation,
    let rep = NSBitmapImageRep(data: tiff),
    let png = rep.representation(using: .png, properties: [:])
else {
    fputs("failed to generate source icon\n", stderr)
    exit(1)
}

try png.write(to: URL(fileURLWithPath: outputPath))
SWIFT

for size in 16 32 128 256 512; do
  sips -z "$size" "$size" "$SOURCE_PNG" --out "$ICONSET_DIR/icon_${size}x${size}.png" >/dev/null
done

for size in 16 32 128 256 512; do
  retina=$((size * 2))
  sips -z "$retina" "$retina" "$SOURCE_PNG" --out "$ICONSET_DIR/icon_${size}x${size}@2x.png" >/dev/null
done

mkdir -p "$(dirname "$OUTPUT_ICNS")"
iconutil -c icns "$ICONSET_DIR" -o "$OUTPUT_ICNS"
rm -rf "$TMP_DIR"
echo "generated icon: $OUTPUT_ICNS"
