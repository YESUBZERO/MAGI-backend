import { create } from "zustand";

export interface DynamicHistoryItem {
  timestamp: string;
  latitude: number;
  longitude: number;
  status: string;
  speed_knots: number;
  course: number;
}

export interface ShipDetails {
  imo: number;
  mmsi: number;
  callsign: string;
  shipname: string;
  ship_type: string;
  total_positions_recorded: number;
  position_history?: DynamicHistoryItem[];
}

interface AISState {
  currentShip: ShipDetails | null;
  isLoading: boolean;
  error: string | null;
  fetchShipByIMO: (imo: number, apiBaseUrl?: string) => Promise<void>;
  fetchShipByMMSI: (mmsi: number, apiBaseUrl?: string) => Promise<void>;
  clearShip: () => void;
}

export const useAISStore = create<AISState>((set) => ({
  currentShip: null,
  isLoading: false,
  error: null,

  fetchShipByIMO: async (imo: number, apiBaseUrl = "http://localhost:8080") => {
    set({ isLoading: true, error: null });
    try {
      const res = await fetch(`${apiBaseUrl}/api/v1/ship/${imo}`);
      if (!res.ok) {
        const errData = await res.json().catch(() => ({ error: "Error consultando buque" }));
        throw new Error(errData.error || `HTTP error: ${res.status}`);
      }
      const data: ShipDetails = await res.json();
      set({ currentShip: data, isLoading: false });
    } catch (err: unknown) {
      set({
        error: err instanceof Error ? err.message : "Error inesperado",
        isLoading: false,
      });
    }
  },

  fetchShipByMMSI: async (mmsi: number, apiBaseUrl = "http://localhost:8080") => {
    set({ isLoading: true, error: null });
    try {
      const res = await fetch(`${apiBaseUrl}/api/v1/ship/mmsi/${mmsi}`);
      if (!res.ok) {
        const errData = await res.json().catch(() => ({ error: "Error consultando buque" }));
        throw new Error(errData.error || `HTTP error: ${res.status}`);
      }
      const data: ShipDetails = await res.json();
      set({ currentShip: data, isLoading: false });
    } catch (err: unknown) {
      set({
        error: err instanceof Error ? err.message : "Error inesperado",
        isLoading: false,
      });
    }
  },

  clearShip: () => set({ currentShip: null, error: null, isLoading: false }),
}));
