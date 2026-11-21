import sys
import os
from PIL import Image
from rembg import new_session, remove

def process_icon(src_jpg, dest_png):
    os.makedirs(os.path.dirname(dest_png), exist_ok=True)
    session = new_session("u2netp")
    raw = Image.open(src_jpg)
    cutout = remove(raw, session=session)
    resized = cutout.resize((256, 256), Image.Resampling.LANCZOS)
    resized.save(dest_png, "PNG")
    print(f"Processed: {src_jpg} -> {dest_png}")

if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Usage: python process_gu_icon.py <src_jpg> <dest_png>")
        sys.exit(1)
    process_icon(sys.argv[1], sys.argv[2])
