#!/usr/bin/env python3
import sys
from pathlib import Path
from PIL import Image, ImageDraw

# Configuration
frame_w, frame_h = 64, 64
rows = {
    "idle": 4,
    "walk": 6,
    "attack": 8,
}
# Colors per row (simple distinct colors)
row_colors = {
    "idle": (200, 200, 255),
    "walk": (200, 255, 200),
    "attack": (255, 200, 200),
}

max_frames = max(rows.values())
cols = max_frames
rows_count = len(rows)

sheet_w = cols * frame_w
sheet_h = rows_count * frame_h
sheet = Image.new("RGBA", (sheet_w, sheet_h), (0,0,0,0))

draw = ImageDraw.Draw(sheet)

for r_idx, (row_name, frame_cnt) in enumerate(rows.items()):
    color = row_colors.get(row_name, (180,180,180))
    for c_idx in range(frame_cnt):
        x0 = c_idx * frame_w
        y0 = r_idx * frame_h
        # draw rectangle
        draw.rectangle([x0, y0, x0+frame_w-1, y0+frame_h-1], fill=color)
        # draw frame number text
        txt = f"{row_name[:1].upper()}{c_idx+1}"
        # simple text (no font loading) – using default
        draw.text((x0+8, y0+8), txt, fill=(0,0,0))

output_path = Path(__file__).parent / "player_sprite.png"
sheet.save(output_path)
print(f"Generated sprite sheet at {output_path}")
