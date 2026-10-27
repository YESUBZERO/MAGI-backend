"use client";

import { useState, useEffect } from "react";
import { useShallow } from "zustand/react/shallow";
import { useShipSearchStore, type ShipItem } from "@/stores/ship-search-store";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { X, Calendar, Activity, Navigation, Compass, Gauge, AlertCircle } from "lucide-react";

interface TelemetryHistoryModalProps {
  ship: ShipItem | null;
  isOpen: boolean;
  onClose: () => void;
}

export function TelemetryHistoryModal({ ship, isOpen, onClose }: TelemetryHistoryModalProps) {
  const {
    telemetryHistory,
    telemetryTotal,
    telemetryPage,
    isLoadingTelemetry,
    getShipTelemetry,
    error,
  } = useShipSearchStore(
    useShallow((state) => ({
      telemetryHistory: state.telemetryHistory,
      telemetryTotal: state.telemetryTotal,
      telemetryPage: state.telemetryPage,
      isLoadingTelemetry: state.isLoadingTelemetry,
      getShipTelemetry: state.getShipTelemetry,
      error: state.error,
    }))
  );

  // Fechas por defecto: últimas 48 horas
  const now = new Date();
  const past = new Date(now.getTime() - 48 * 3600 * 1000);

  const [from, setFrom] = useState(past.toISOString().slice(0, 16));
  const [to, setTo] = useState(now.toISOString().slice(0, 16));

  useEffect(() => {
    if (ship && isOpen) {
      const fromISO = new Date(from).toISOString();
      const toISO = new Date(to).toISOString();
      void getShipTelemetry(ship.mmsi, fromISO, toISO, 1);
    }
  }, [ship, isOpen]);

  if (!isOpen || !ship) return null;

  const handleQuery = () => {
    const fromISO = new Date(from).toISOString();
    const toISO = new Date(to).toISOString();
    void getShipTelemetry(ship.mmsi, fromISO, toISO, 1);
  };

  const handlePageChange = (newPage: number) => {
    const fromISO = new Date(from).toISOString();
    const toISO = new Date(to).toISOString();
    void getShipTelemetry(ship.mmsi, fromISO, toISO, newPage);
  };

  const totalPages = Math.ceil(telemetryTotal / 50) || 1;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-xs p-4">
      <div className="bg-background border rounded-lg shadow-xl w-full max-w-4xl max-h-[90vh] flex flex-col overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        {/* Cabecera del Modal */}
        <div className="flex items-center justify-between border-b px-5 py-4 bg-muted/30">
          <div className="flex items-center gap-2">
            <Activity className="w-5 h-5 text-emerald-500" />
            <div>
              <h2 className="text-base font-bold text-foreground">
                Historial de Telemetría Dinámica - {ship.shipname}
              </h2>
              <p className="text-xs text-muted-foreground">
                MMSI: {ship.mmsi} | IMO: {ship.imo || "N/A"} | Tipo: {ship.ship_type || "N/A"}
              </p>
            </div>
          </div>
          <Button variant="ghost" size="icon-sm" onClick={onClose}>
            <X className="w-4 h-4" />
          </Button>
        </div>

        {/* Filtros de Rango Temporal ISO8601 */}
        <div className="p-4 border-b bg-muted/10 grid grid-cols-1 sm:grid-cols-3 gap-3 items-end">
          <div className="space-y-1">
            <Label htmlFor="date-from" className="text-xs font-medium flex items-center gap-1">
              <Calendar className="w-3 h-3 text-blue-500" />
              Desde (Fecha y Hora)
            </Label>
            <Input
              id="date-from"
              type="datetime-local"
              value={from}
              onChange={(e) => setFrom(e.target.value)}
            />
          </div>

          <div className="space-y-1">
            <Label htmlFor="date-to" className="text-xs font-medium flex items-center gap-1">
              <Calendar className="w-3 h-3 text-blue-500" />
              Hasta (Fecha y Hora)
            </Label>
            <Input
              id="date-to"
              type="datetime-local"
              value={to}
              onChange={(e) => setTo(e.target.value)}
            />
          </div>

          <Button onClick={handleQuery} disabled={isLoadingTelemetry} size="sm">
            {isLoadingTelemetry ? "Cargando..." : "Consultar Rango"}
          </Button>
        </div>

        {/* Contenido / Puntos de Telemetría */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {error && (
            <div className="flex items-center gap-2 p-3 text-xs text-destructive bg-destructive/10 rounded-md border border-destructive/30">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {isLoadingTelemetry ? (
            <div className="py-12 text-center text-xs text-muted-foreground animate-pulse">
              Consultando telemetría dinámica marítima...
            </div>
          ) : telemetryHistory.length === 0 ? (
            <div className="py-12 text-center text-xs text-muted-foreground">
              No se registraron posiciones para este buque en el período seleccionado.
            </div>
          ) : (
            <div className="border rounded-md overflow-hidden">
              <table className="w-full text-left text-xs">
                <thead className="bg-muted/50 border-b uppercase text-muted-foreground font-semibold">
                  <tr>
                    <th className="px-3 py-2">Fecha / Hora</th>
                    <th className="px-3 py-2">Coordenadas</th>
                    <th className="px-3 py-2">Velocidad</th>
                    <th className="px-3 py-2">Rumbo</th>
                    <th className="px-3 py-2">Estado de Navegación</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {telemetryHistory.map((item, idx) => (
                    <tr key={idx} className="hover:bg-muted/20">
                      <td className="px-3 py-2 font-mono whitespace-nowrap">
                        {new Date(item.timestamp).toLocaleString()}
                      </td>
                      <td className="px-3 py-2 font-mono flex items-center gap-1">
                        <Navigation className="w-3 h-3 text-blue-500" />
                        {item.latitude.toFixed(4)}°, {item.longitude.toFixed(4)}°
                      </td>
                      <td className="px-3 py-2 font-medium">
                        <span className="flex items-center gap-1">
                          <Gauge className="w-3 h-3 text-amber-500" />
                          {item.speed_knots.toFixed(1)} kn
                        </span>
                      </td>
                      <td className="px-3 py-2 font-mono">
                        <span className="flex items-center gap-1">
                          <Compass className="w-3 h-3 text-teal-500" />
                          {item.course.toFixed(1)}°
                        </span>
                      </td>
                      <td className="px-3 py-2">
                        <span className="px-2 py-0.5 rounded text-[10px] bg-muted border">
                          {item.status || "Under way"}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {/* Paginación y Pie del Modal */}
        <div className="border-t px-4 py-3 bg-muted/20 flex items-center justify-between text-xs">
          <span className="text-muted-foreground">
            Total de spots: <strong className="text-foreground">{telemetryTotal}</strong>
          </span>
          {totalPages > 1 && (
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="xs"
                onClick={() => handlePageChange(telemetryPage - 1)}
                disabled={telemetryPage <= 1 || isLoadingTelemetry}
              >
                Anterior
              </Button>
              <span>{telemetryPage} / {totalPages}</span>
              <Button
                variant="outline"
                size="xs"
                onClick={() => handlePageChange(telemetryPage + 1)}
                disabled={telemetryPage >= totalPages || isLoadingTelemetry}
              >
                Siguiente
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
