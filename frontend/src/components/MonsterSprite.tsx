import React from "react";
import { getMonsterArt } from "@/lib/assets/MonsterArt";

interface MonsterSpriteProps {
  templateId: string;
  alt: string;
  className?: string;
}

export const MonsterSprite: React.FC<MonsterSpriteProps> = ({ templateId, alt, className = "" }) => {
  const art = getMonsterArt(templateId);
  if (!art) {
    return <img src="/assets/world/spirit-beast.webp" alt={alt} className={className} />;
  }

  const rows = art.rows;
  const positionX = art.columns > 1 ? (art.column / (art.columns - 1)) * 100 : 0;
  const positionY = rows > 1 ? (art.row / (rows - 1)) * 100 : 0;

  return (
    <span
      role="img"
      aria-label={alt}
      className={`block bg-no-repeat ${className}`}
      style={{
        backgroundImage: `url(${art.src})`,
        backgroundSize: `${art.columns * 100}% ${rows * 100}%`,
        backgroundPosition: `${positionX}% ${positionY}%`,
      }}
    />
  );
};
