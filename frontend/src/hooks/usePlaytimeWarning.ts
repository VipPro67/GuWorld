"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import { useGameStore } from "@/store/useGameStore";

const STORAGE_KEY = "gu_playtime_session_start";
const WARNED_KEY = "gu_playtime_warned_milestones";

export function usePlaytimeWarning(isLoggedIn: boolean) {
  const [playtimeMinutes, setPlaytimeMinutes] = useState<number>(0);
  const [show180Modal, setShow180Modal] = useState<boolean>(false);
  const addToast = useGameStore((state) => state.addToast);
  const warnedMilestonesRef = useRef<Set<number>>(new Set());

  // Initialize or restore session
  useEffect(() => {
    if (!isLoggedIn) {
      if (typeof window !== "undefined") {
        sessionStorage.removeItem(STORAGE_KEY);
        sessionStorage.removeItem(WARNED_KEY);
      }
      setPlaytimeMinutes(0);
      setShow180Modal(false);
      warnedMilestonesRef.current.clear();
      return;
    }

    if (typeof window === "undefined") return;

    let startTime = sessionStorage.getItem(STORAGE_KEY);
    if (!startTime) {
      startTime = Date.now().toString();
      sessionStorage.setItem(STORAGE_KEY, startTime);
    }

    const savedWarned = sessionStorage.getItem(WARNED_KEY);
    if (savedWarned) {
      try {
        const arr = JSON.parse(savedWarned);
        if (Array.isArray(arr)) {
          warnedMilestonesRef.current = new Set(arr);
        }
      } catch {
        // ignore parse error
      }
    }

    const startTimestamp = parseInt(startTime, 10) || Date.now();

    const interval = setInterval(() => {
      const elapsedMinutes = Math.floor((Date.now() - startTimestamp) / 60000);
      setPlaytimeMinutes(elapsedMinutes);

      // Check milestones
      if (elapsedMinutes >= 60 && !warnedMilestonesRef.current.has(60)) {
        warnedMilestonesRef.current.add(60);
        saveWarned();
        addToast(
          "🔔 Đạo hữu đã tu luyện liên tục 60 phút. Hãy chớp mắt và uống nước để điều hòa khí tức!",
          "info"
        );
      }

      if (elapsedMinutes >= 120 && !warnedMilestonesRef.current.has(120)) {
        warnedMilestonesRef.current.add(120);
        saveWarned();
        addToast(
          "⚠️ Đạo hữu đã nhập định 120 phút. Vận chuyển chân nguyên quá lâu có thể tổn hao tinh thần!",
          "info"
        );
      }

      // 180 minutes milestone & every 60 minutes thereafter (Article 39 & 40 Decree 147/2024/NĐ-CP)
      if (elapsedMinutes >= 180) {
        const currentMilestone = Math.floor(elapsedMinutes / 60) * 60;
        if (!warnedMilestonesRef.current.has(currentMilestone)) {
          warnedMilestonesRef.current.add(currentMilestone);
          saveWarned();
          setShow180Modal(true);
        }
      }
    }, 15000); // Check every 15s

    const saveWarned = () => {
      sessionStorage.setItem(
        WARNED_KEY,
        JSON.stringify(Array.from(warnedMilestonesRef.current))
      );
    };

    return () => clearInterval(interval);
  }, [isLoggedIn, addToast]);

  const dismiss180Modal = useCallback(() => {
    setShow180Modal(false);
  }, []);

  return {
    playtimeMinutes,
    show180Modal,
    dismiss180Modal,
  };
}
