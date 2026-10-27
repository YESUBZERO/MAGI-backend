import { create } from "zustand";
import { apiClient } from "@/lib/api-client";

export interface ShipItem {
  id: number;
  msg_type: number;
  imo: number;
  mmsi: number;
  callsign: string;
  shipname: string;
  ship_type: string;
  created_at: string;
  updated_at: string;
}

export interface TelemetryPoint {
  timestamp: string;
  latitude: number;
  longitude: number;
  status: string;
  speed_knots: number;
  course: number;
}

export interface ShipSearchFilterState {
  q?: string;
  imo?: number;
  mmsi?: number;
  shipType?: string;
}

interface ShipSearchState {
  ships: ShipItem[];
  total: number;
  page: number;
  pageSize: number;
  filters: ShipSearchFilterState;
  selectedShip: ShipItem | null;
  telemetryHistory: TelemetryPoint[];
  telemetryTotal: number;
  telemetryPage: number;
  isLoading: boolean;
  isLoadingTelemetry: boolean;
  error: string | null;

  // Acciones
  setFilters: (filters: Partial<ShipSearchFilterState>) => void;
  clearFilters: () => void;
  searchShips: (page?: number) => Promise<void>;
  setSelectedShip: (ship: ShipItem | null) => void;
  getShipTelemetry: (mmsi: number, from: string, to: string, page?: number) => Promise<void>;
  clearTelemetry: () => void;
}

export const useShipSearchStore = create<ShipSearchState>((set, get) => ({
  ships: [],
  total: 0,
  page: 1,
  pageSize: 20,
  filters: {},
  selectedShip: null,
  telemetryHistory: [],
  telemetryTotal: 0,
  telemetryPage: 1,
  isLoading: false,
  isLoadingTelemetry: false,
  error: null,

  setFilters: (newFilters) => {
    set((state) => ({
      filters: { ...state.filters, ...newFilters },
      page: 1,
    }));
  },

  clearFilters: () => {
    set({
      filters: {},
      page: 1,
      ships: [],
      total: 0,
      error: null,
    });
  },

  searchShips: async (targetPage) => {
    const { filters, pageSize, page } = get();
    const currentPage = targetPage !== undefined ? targetPage : page;
    const offset = (currentPage - 1) * pageSize;

    set({ isLoading: true, error: null, page: currentPage });

    try {
      const params = new URLSearchParams();
      if (filters.q) params.set("q", filters.q.trim());
      if (filters.imo) params.set("imo", filters.imo.toString());
      if (filters.mmsi) params.set("mmsi", filters.mmsi.toString());
      if (filters.shipType) params.set("ship_type", filters.shipType);
      params.set("limit", pageSize.toString());
      params.set("offset", offset.toString());

      const data = await apiClient<{
        total: number;
        limit: number;
        offset: number;
        ships: ShipItem[];
      }>(`/api/v1/ships?${params.toString()}`);

      set({
        ships: data.ships || [],
        total: data.total || 0,
        isLoading: false,
      });
    } catch (err: unknown) {
      set({
        error: err instanceof Error ? err.message : "Error buscando buques",
        isLoading: false,
      });
    }
  },

  setSelectedShip: (ship) => {
    set({ selectedShip: ship, telemetryHistory: [], telemetryTotal: 0 });
  },

  getShipTelemetry: async (mmsi: number, from: string, to: string, targetPage = 1) => {
    const limit = 50;
    const offset = (targetPage - 1) * limit;

    set({ isLoadingTelemetry: true, error: null, telemetryPage: targetPage });

    try {
      const params = new URLSearchParams({
        from,
        to,
        limit: limit.toString(),
        offset: offset.toString(),
      });

      const data = await apiClient<{
        mmsi: number;
        from: string;
        to: string;
        total: number;
        telemetry: TelemetryPoint[];
      }>(`/api/v1/ships/${mmsi}/telemetry?${params.toString()}`);

      set({
        telemetryHistory: data.telemetry || [],
        telemetryTotal: data.total || 0,
        isLoadingTelemetry: false,
      });
    } catch (err: unknown) {
      set({
        error: err instanceof Error ? err.message : "Error consultando telemetría",
        isLoadingTelemetry: false,
      });
    }
  },

  clearTelemetry: () => {
    set({ telemetryHistory: [], telemetryTotal: 0, selectedShip: null });
  },
}));
