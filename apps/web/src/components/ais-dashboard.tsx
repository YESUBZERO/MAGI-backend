"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { useAISStore } from "@/stores/ais-store";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { TurnstileWidget } from "@/components/turnstile-widget";

interface AISQueryFormValues {
  queryType: "mmsi" | "imo";
  identifier: number;
}

export function AISDashboard() {
  const { currentShip, isLoading, error, fetchShipByIMO, fetchShipByMMSI, clearShip } = useAISStore();
  const [turnstileToken, setTurnstileToken] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<AISQueryFormValues>({
    defaultValues: {
      queryType: "mmsi",
      identifier: 244670000,
    },
  });

  const queryType = watch("queryType");

  const onSubmit = async (data: AISQueryFormValues) => {
    const id = Number(data.identifier);
    if (!id || id <= 0) return;

    if (data.queryType === "imo") {
      await fetchShipByIMO(id);
    } else {
      await fetchShipByMMSI(id);
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="text-xl">Búsqueda y Telemetría AIS</CardTitle>
          <CardDescription>
            Consulta información estática del buque y su historial dinámico de telemetría marítima.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <div className="space-y-2">
                <Label htmlFor="queryType">Tipo de Identificador</Label>
                <select
                  id="queryType"
                  {...register("queryType")}
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                >
                  <option value="mmsi">MMSI (9 dígitos)</option>
                  <option value="imo">IMO (7 dígitos)</option>
                </select>
              </div>

              <div className="space-y-2 sm:col-span-2">
                <Label htmlFor="identifier">
                  {queryType === "mmsi" ? "Número MMSI" : "Número IMO"}
                </Label>
                <div className="flex gap-2">
                  <Input
                    id="identifier"
                    type="number"
                    placeholder={queryType === "mmsi" ? "Ej. 244670000" : "Ej. 9241061"}
                    {...register("identifier", {
                      required: "El identificador es requerido",
                      valueAsNumber: true,
                    })}
                  />
                  <Button type="submit" disabled={isLoading}>
                    {isLoading ? "Consultando..." : "Consultar"}
                  </Button>
                </div>
                {errors.identifier && (
                  <p className="text-xs text-destructive">{errors.identifier.message}</p>
                )}
              </div>
            </div>

            <div className="pt-2">
              <TurnstileWidget
                onSuccess={(token) => setTurnstileToken(token)}
                className="my-1"
              />
            </div>
          </form>
        </CardContent>
      </Card>

      {error && (
        <Card className="border-destructive/50 bg-destructive/10">
          <CardContent className="pt-6">
            <p className="text-sm font-medium text-destructive">❌ {error}</p>
          </CardContent>
        </Card>
      )}

      {currentShip && (
        <div className="space-y-6">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <div>
                <CardTitle className="text-2xl">{currentShip.shipname || "UNKNOWN"}</CardTitle>
                <CardDescription>
                  Tipo: {currentShip.ship_type || "No clasificado"} | Callsign: {currentShip.callsign || "N/A"}
                </CardDescription>
              </div>
              <Button variant="outline" size="sm" onClick={clearShip}>
                Limpiar
              </Button>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
                <div className="rounded-lg border p-3">
                  <p className="text-xs text-muted-foreground">IMO</p>
                  <p className="text-lg font-semibold">{currentShip.imo || "N/A"}</p>
                </div>
                <div className="rounded-lg border p-3">
                  <p className="text-xs text-muted-foreground">MMSI</p>
                  <p className="text-lg font-semibold">{currentShip.mmsi}</p>
                </div>
                <div className="rounded-lg border p-3">
                  <p className="text-xs text-muted-foreground">Posiciones Registradas</p>
                  <p className="text-lg font-semibold">{currentShip.total_positions_recorded}</p>
                </div>
                <div className="rounded-lg border p-3">
                  <p className="text-xs text-muted-foreground">Estado</p>
                  <p className="text-lg font-semibold text-emerald-600 dark:text-emerald-400">Activo</p>
                </div>
              </div>

              {currentShip.position_history && currentShip.position_history.length > 0 && (
                <div className="mt-6">
                  <h3 className="mb-3 text-sm font-semibold">Historial Dinámico de Telemetría (Últimos Spots)</h3>
                  <div className="overflow-x-auto rounded-md border">
                    <table className="w-full text-left text-sm">
                      <thead className="border-b bg-muted/50 text-xs uppercase text-muted-foreground">
                        <tr>
                          <th className="px-4 py-2">Fecha / Hora</th>
                          <th className="px-4 py-2">Latitud</th>
                          <th className="px-4 py-2">Longitud</th>
                          <th className="px-4 py-2">Velocidad (kn)</th>
                          <th className="px-4 py-2">Rumbo (°)</th>
                          <th className="px-4 py-2">Estado</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y">
                        {currentShip.position_history.map((spot, idx) => (
                          <tr key={idx} className="hover:bg-muted/30">
                            <td className="px-4 py-2 whitespace-nowrap font-mono text-xs">
                              {new Date(spot.timestamp).toLocaleString()}
                            </td>
                            <td className="px-4 py-2 font-mono text-xs">{spot.latitude.toFixed(4)}</td>
                            <td className="px-4 py-2 font-mono text-xs">{spot.longitude.toFixed(4)}</td>
                            <td className="px-4 py-2">{spot.speed_knots.toFixed(1)}</td>
                            <td className="px-4 py-2">{spot.course.toFixed(1)}°</td>
                            <td className="px-4 py-2 text-xs">{spot.status || "Under way"}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
