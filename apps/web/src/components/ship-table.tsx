"use client";

import { useShallow } from "zustand/react/shallow";
import { useShipSearchStore, type ShipItem } from "@/stores/ship-search-store";
import { Button } from "@/components/ui/button";
import { Activity, Download, Ship, ChevronLeft, ChevronRight } from "lucide-react";

interface ShipTableProps {
  onOpenTelemetry: (ship: ShipItem) => void;
  onExportShip?: (ship: ShipItem) => void;
}

export function ShipTable({ onOpenTelemetry, onExportShip }: ShipTableProps) {
  const { ships, total, page, pageSize, searchShips, isLoading } = useShipSearchStore(
    useShallow((state) => ({
      ships: state.ships,
      total: state.total,
      page: state.page,
      pageSize: state.pageSize,
      searchShips: state.searchShips,
      isLoading: state.isLoading,
    }))
  );

  const totalPages = Math.ceil(total / pageSize) || 1;

  if (ships.length === 0 && !isLoading) {
    return (
      <div className="rounded-lg border border-dashed p-8 text-center bg-muted/20">
        <Ship className="mx-auto h-10 w-10 text-muted-foreground/60 mb-3" />
        <p className="text-sm font-medium text-foreground">No se encontraron buques</p>
        <p className="text-xs text-muted-foreground mt-1">
          Ajusta los filtros de búsqueda o ingresa un nombre/MMSI para consultar la flota.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between text-xs text-muted-foreground">
        <span>
          Mostrando {ships.length > 0 ? (page - 1) * pageSize + 1 : 0} -{" "}
          {Math.min(page * pageSize, total)} de {total} buques registrados
        </span>
        <span>Página {page} de {totalPages}</span>
      </div>

      <div className="rounded-lg border overflow-hidden shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead className="bg-muted/60 border-b uppercase font-semibold text-muted-foreground">
              <tr>
                <th className="px-4 py-3">Nombre</th>
                <th className="px-4 py-3">MMSI</th>
                <th className="px-4 py-3">IMO</th>
                <th className="px-4 py-3">Callsign</th>
                <th className="px-4 py-3">Tipo</th>
                <th className="px-4 py-3 text-right">Acciones</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {ships.map((ship) => (
                <tr key={ship.mmsi} className="hover:bg-muted/40 transition-colors">
                  <td className="px-4 py-3 font-medium text-foreground flex items-center gap-2">
                    <Ship className="w-3.5 h-3.5 text-blue-500 shrink-0" />
                    <span className="font-semibold">{ship.shipname || "UNKNOWN"}</span>
                  </td>
                  <td className="px-4 py-3 font-mono">{ship.mmsi}</td>
                  <td className="px-4 py-3 font-mono">{ship.imo || "N/A"}</td>
                  <td className="px-4 py-3 font-mono uppercase">{ship.callsign || "-"}</td>
                  <td className="px-4 py-3">
                    <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-medium bg-blue-50 dark:bg-blue-950 text-blue-700 dark:text-blue-300 border border-blue-200 dark:border-blue-800">
                      {ship.ship_type || "No clasificado"}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-right space-x-1 whitespace-nowrap">
                    <Button
                      variant="outline"
                      size="xs"
                      onClick={() => onOpenTelemetry(ship)}
                      className="gap-1"
                    >
                      <Activity className="w-3 h-3 text-emerald-500" />
                      Telemetría
                    </Button>
                    {onExportShip && (
                      <Button
                        variant="secondary"
                        size="xs"
                        onClick={() => onExportShip(ship)}
                        className="gap-1"
                      >
                        <Download className="w-3 h-3" />
                        Exportar
                      </Button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Paginación */}
      {totalPages > 1 && (
        <div className="flex items-center justify-end gap-2 pt-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => void searchShips(page - 1)}
            disabled={page <= 1 || isLoading}
          >
            <ChevronLeft className="w-3.5 h-3.5 mr-1" />
            Anterior
          </Button>
          <span className="text-xs font-medium px-2">
            {page} / {totalPages}
          </span>
          <Button
            variant="outline"
            size="sm"
            onClick={() => void searchShips(page + 1)}
            disabled={page >= totalPages || isLoading}
          >
            Siguiente
            <ChevronRight className="w-3.5 h-3.5 ml-1" />
          </Button>
        </div>
      )}
    </div>
  );
}
