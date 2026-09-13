import type { StateCreator } from "zustand";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import type { PlayerPet } from "@/store/useGameStore";

export interface PetsSlice {
  playerPets: PlayerPet[];
  playerPetsLoaded: boolean;
  fetchPlayerPets: () => Promise<void>;
  feedPet: (petId: string, itemId: string) => Promise<boolean>;
  setPetFormation: (petIds: string[]) => Promise<boolean>;
  setPetSleep: (petId: string, sleeping: boolean) => Promise<boolean>;
  releasePet: (petId: string) => Promise<boolean>;
  mergePets: (targetId: string, sourceIds: string[]) => Promise<boolean>;
  captureMonster: () => Promise<{ captured: boolean; message: string } | null>;
}

export const createPetsSlice: StateCreator<any, any, any, PetsSlice> = (set, get) => ({
  playerPets: [],
  playerPetsLoaded: false,

  fetchPlayerPets: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/pets`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      if (response.ok) {
        set({ playerPets: data.pets || [], playerPetsLoaded: true });
      }
    } catch {}
  },

  feedPet: async (petId, itemId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/pets/feed`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ pet_id: petId, item_id: itemId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cho ăn thất bại");
      get().addToast(data.message || `Đã cho ăn +${data.restored} hunger`, "success");
      await get().fetchPlayerPets();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  setPetFormation: async (petIds) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/pets/formation`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ slots: petIds }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cập nhật đội hình thất bại");
      get().addToast("Đội hình đã cập nhật", "success");
      await get().fetchPlayerPets();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  setPetSleep: async (petId, sleeping) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/pets/sleep`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ pet_id: petId, sleeping }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cập nhật trạng thái thất bại");
      await get().fetchPlayerPets();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  releasePet: async (petId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/pets/release`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ pet_id: petId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Thả pet thất bại");
      get().addToast("Đã thả pet", "success");
      await get().fetchPlayerPets();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  mergePets: async (targetId, sourceIds) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/pets/merge`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ target_id: targetId, source_ids: sourceIds }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Gộp bầy thất bại");
      get().addToast(data.message || `Đã gộp thành ${data.count} cá thể!`, "success");
      await get().fetchPlayerPets();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  captureMonster: async () => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/combat/capture`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ player_id: player.id }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Bắt thất bại");
      // Re-sync inventory from server
      await get().initializeSession();
      if (data.captured) {
        get().addToast(data.message || "Bắt thành công!", "success");
        await get().fetchPlayerPets();
      } else {
        get().addToast(data.message || "Bắt thất bại!", "error");
      }
      return { captured: data.captured, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return null;
    } finally { set({ actionLoading: false }); }
  },
});
