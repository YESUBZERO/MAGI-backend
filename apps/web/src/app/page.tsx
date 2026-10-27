"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import { ShipSearchBar } from "@/components/ship-search-bar";
import { ShipTable } from "@/components/ship-table";
import { TelemetryHistoryModal } from "@/components/telemetry-history-modal";
import { GraphBackendStatus } from "@/components/graph-backend-status";
import { useShipSearchStore, type ShipItem } from "@/stores/ship-search-store";
import { apiClient, ApiError } from "@/lib/api-client";

export default function Home() {
  const [modalShip, setModalShip] = useState<ShipItem | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const { searchShips } = useShipSearchStore();

  useEffect(() => {
    // Carga inicial de la flota registrada
    void searchShips(1);
  }, []);

  const handleOpenTelemetry = (ship: ShipItem) => {
    setModalShip(ship);
    setIsModalOpen(true);
  };

  const handleCloseTelemetry = () => {
    setIsModalOpen(false);
    setModalShip(null);
  };

  const handleExportShip = async (ship: ShipItem) => {
    try {
      const data = await apiClient<{ message: string; job_id: string }>(
        "/api/v1/exports",
        {
          method: "POST",
          body: JSON.stringify({
            mmsi: ship.mmsi,
            format: "csv",
          }),
        }
      );
      toast.success(
        `Exportación encolada para ${ship.shipname}. Tarea: ${data.job_id.slice(0, 8)}...`
      );
    } catch (err: unknown) {
      if (err instanceof ApiError && err.status === 403) {
        toast.error("Permiso denegado: Se requiere el rol 'ais:export' para exportar datos.");
      } else if (err instanceof ApiError && err.status === 401) {
        toast.error("Sesión requerida: Inicia sesión con Zitadel para exportar datos.");
      } else {
        toast.error(err instanceof Error ? err.message : "Error al solicitar exportación");
      }
    }
  };

  return (
    <div className="container mx-auto max-w-6xl px-4 py-8 space-y-8">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">MAGI Fleet & Telemetry Portal</h1>
        <p className="text-muted-foreground mt-1 text-sm">
          Búsqueda facetada de buques y telemetría histórica de alta resolución con autenticación Zitadel OIDC.
        </p>
      </div>

      <ShipSearchBar />

      <section className="space-y-3">
        <h2 className="text-base font-semibold text-foreground">Resultados de Embarcaciones</h2>
        <ShipTable
          onOpenTelemetry={handleOpenTelemetry}
          onExportShip={handleExportShip}
        />
      </section>

      <section className="rounded-lg border p-4 bg-card shadow-xs">
        <h3 className="mb-2 font-medium text-xs text-muted-foreground uppercase tracking-wider">
          Estado del Backend Gin
        </h3>
        <GraphBackendStatus />
      </section>

      <TelemetryHistoryModal
        ship={modalShip}
        isOpen={isModalOpen}
        onClose={handleCloseTelemetry}
      />
    </div>
  );
}
