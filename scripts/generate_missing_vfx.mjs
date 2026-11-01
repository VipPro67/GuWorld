import fs from "node:fs";
import path from "node:path";
import { execFileSync } from "node:child_process";

const OUT_DIR = "frontend/public/assets/vfx";
fs.mkdirSync(OUT_DIR, { recursive: true });

function generateArcAtlas(name, colorCenter, colorGlow, sparkColor) {
  const tmpDir = path.join("/tmp", name);
  fs.mkdirSync(tmpDir, { recursive: true });

  const numFrames = 6;
  const frameWidth = 128;
  const frameHeight = 128;
  const frameFiles = [];

  for (let i = 0; i < numFrames; i++) {
    const frameFile = path.join(tmpDir, `frame_${i}.png`);
    frameFiles.push(frameFile);

    const progress = i / (numFrames - 1);
    const radius = 24 + progress * 28;
    const alpha = i === 0 ? 0.8 : i === numFrames - 1 ? 0.3 : 1.0;
    const strokeWidth = i < 3 ? 5 + i * 2 : Math.max(2, 9 - (i - 2) * 2);

    // Draw crescent arc via ImageMagick
    // We draw an arc and a blurred glow layer
    execFileSync("magick", [
      "-size", `${frameWidth}x${frameHeight}`, "xc:none",
      "-stroke", colorGlow,
      "-strokewidth", String(strokeWidth + 4),
      "-fill", "none",
      "-draw", `arc 20,20 ${frameWidth - 20},${frameHeight - 20} 220 320`,
      "-blur", "0x3",
      "-stroke", colorCenter,
      "-strokewidth", String(strokeWidth),
      "-draw", `arc 20,20 ${frameWidth - 20},${frameHeight - 20} 220 320`,
      "-stroke", sparkColor,
      "-strokewidth", "2",
      "-draw", `circle ${frameWidth / 2},${20 + progress * 10} ${frameWidth / 2 + 2},${20 + progress * 10}`,
      frameFile
    ]);
  }

  // Combine into horizontal strip
  const outPng = path.join(OUT_DIR, `${name}.png`);
  execFileSync("magick", [...frameFiles, "+append", outPng]);

  // Generate Phaser Hash JSON
  const framesObj = {};
  for (let i = 0; i < numFrames; i++) {
    framesObj[String(i)] = {
      frame: { x: i * frameWidth, y: 0, w: frameWidth, h: frameHeight },
      rotated: false,
      trimmed: false,
      spriteSourceSize: { x: 0, y: 0, w: frameWidth, h: frameHeight },
      sourceSize: { w: frameWidth, h: frameHeight }
    };
  }

  const jsonMeta = {
    frames: framesObj,
    meta: {
      image: `${name}.png`,
      size: { w: numFrames * frameWidth, h: frameHeight },
      scale: "1"
    }
  };

  fs.writeFileSync(path.join(OUT_DIR, `${name}.json`), JSON.stringify(jsonMeta, null, 2));
  console.log(`Generated atlas: ${name}`);
}

function generateBurstAtlas(name, colorCore, colorOuter) {
  const tmpDir = path.join("/tmp", name);
  fs.mkdirSync(tmpDir, { recursive: true });

  const numFrames = 6;
  const frameWidth = 128;
  const frameHeight = 128;
  const frameFiles = [];

  for (let i = 0; i < numFrames; i++) {
    const frameFile = path.join(tmpDir, `frame_${i}.png`);
    frameFiles.push(frameFile);

    const r = 8 + i * 9;
    const strokeW = Math.max(1, 8 - i);
    execFileSync("magick", [
      "-size", `${frameWidth}x${frameHeight}`, "xc:none",
      "-fill", colorOuter,
      "-stroke", colorCore,
      "-strokewidth", String(strokeW),
      "-draw", `circle 64,64 ${64 + r},64`,
      "-blur", "0x2",
      "-fill", "none",
      "-stroke", "#ffffff",
      "-strokewidth", "2",
      "-draw", `circle 64,64 ${64 + Math.max(2, r - 6)},64`,
      frameFile
    ]);
  }

  const outPng = path.join(OUT_DIR, `${name}.png`);
  execFileSync("magick", [...frameFiles, "+append", outPng]);

  const framesObj = {};
  for (let i = 0; i < numFrames; i++) {
    framesObj[String(i)] = {
      frame: { x: i * frameWidth, y: 0, w: frameWidth, h: frameHeight },
      rotated: false,
      trimmed: false,
      spriteSourceSize: { x: 0, y: 0, w: frameWidth, h: frameHeight },
      sourceSize: { w: frameWidth, h: frameHeight }
    };
  }

  const jsonMeta = {
    frames: framesObj,
    meta: {
      image: `${name}.png`,
      size: { w: numFrames * frameWidth, h: frameHeight },
      scale: "1"
    }
  };

  fs.writeFileSync(path.join(OUT_DIR, `${name}.json`), JSON.stringify(jsonMeta, null, 2));
  console.log(`Generated burst atlas: ${name}`);
}

// 1. Nguyệt Quang Cổ (Moonblade jade-cyan crescent)
generateArcAtlas("fx_gu_nguyet_quang", "#67e8f9", "#06b6d4", "#a5f3fc");

// 2. Huyết Nguyệt Cổ (Blood moon ruby-crimson crescent)
generateArcAtlas("fx_gu_huyet_nguyet", "#ef4444", "#991b1b", "#fecaca");

// 3. Huyết Đạo Bạo Phát (Blood explosion splash)
generateBurstAtlas("fx_gu_huyet_bao", "#dc2626", "#7f1d1d");

// 4. Bạch Cốt Mâu (Bone spike burst)
generateBurstAtlas("fx_gu_bach_cot", "#f1f5f9", "#64748b");
