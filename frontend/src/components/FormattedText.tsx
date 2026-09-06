"use client";

import React from "react";
import { Num } from "./Num";

interface FormattedTextProps {
  text?: string;
  className?: string;
}

export function FormattedText({ text, className }: FormattedTextProps) {
  if (!text) return null;

  // Split text by:
  // 1. Bracketed skill/Gu names: \[[^\]]+\]
  // 2. 4+ digit numbers: \b\d{4,}\b
  // 3. Key terms: Khiên Ảo|Khiên|khiên|sát thương thực|sát thương|hồi phục|Hồi Sinh|do Đạo Ngân|Đạo Ngân|\bHP\b|\bmáu\b
  const regex = /(\[[^\]]+\]|\b\d{4,}\b|Khiên Ảo|sát thương thực|sát thương|hồi phục|Hồi Sinh|do Đạo Ngân|Đạo Ngân|\bHP\b|\bmáu\b)/gi;
  const rawTokens = text.split(regex);

  // Helper to determine context for numbers
  const getContext = (index: number, tokens: string[]) => {
    // Look ahead up to 4 tokens for indicator keywords
    for (let i = index + 1; i < Math.min(tokens.length, index + 5); i++) {
      const tok = tokens[i].toLowerCase();
      if (tok.includes("khiên")) return "shield";
      if (tok.includes("sát thương")) return "damage";
      if (tok.includes("hp") || tok.includes("hồi") || tok.includes("máu")) return "heal";
    }
    // Look back up to 4 tokens if lookahead didn't specify
    for (let i = index - 1; i >= Math.max(0, index - 5); i--) {
      const tok = tokens[i].toLowerCase();
      if (tok.includes("khiên")) return "shield";
      if (tok.includes("gây") || tok.includes("chịu") || tok.includes("phản") || tok.includes("sát thương")) return "damage";
      if (tok.includes("hồi") || tok.includes("hút")) return "heal";
    }
    return "default";
  };

  return (
    <span className={className}>
      {rawTokens.map((token, index) => {
        if (!token) return null;

        // 1. Bracketed skill/Gu name
        if (/^\[[^\]]+\]$/.test(token)) {
          return (
            <span key={index} className="text-purple-300 font-semibold drop-shadow-sm px-0.5">
              {token}
            </span>
          );
        }

        // 2. 4+ digit number
        if (/^\d{4,}$/.test(token)) {
          const num = Number(token);
          if (Number.isFinite(num)) {
            const ctx = getContext(index, rawTokens);
            let numColor = "text-amber-300 font-bold";
            if (ctx === "shield") numColor = "text-cyan-400 font-bold";
            if (ctx === "damage") numColor = "text-rose-400 font-bold";
            if (ctx === "heal") numColor = "text-emerald-400 font-bold";

            return <Num key={index} value={num} className={numColor} />;
          }
        }

        // 3. Keywords formatting
        const lower = token.toLowerCase();

        if (lower === "khiên ảo" || lower === "khiên") {
          return (
            <span key={index} className="text-cyan-400 font-semibold">
              {token}
            </span>
          );
        }

        if (lower === "sát thương" || lower === "sát thương thực") {
          return (
            <span key={index} className="text-rose-400 font-semibold">
              {token}
            </span>
          );
        }

        if (lower === "hồi phục" || lower === "hồi sinh" || lower === "hp") {
          return (
            <span key={index} className="text-emerald-400 font-semibold">
              {token}
            </span>
          );
        }

        if (lower === "do đạo ngân" || lower === "đạo ngân") {
          return (
            <span key={index} className="text-indigo-300 font-medium">
              {token}
            </span>
          );
        }

        // 4. Default plain text
        return <React.Fragment key={index}>{token}</React.Fragment>;
      })}
    </span>
  );
}
