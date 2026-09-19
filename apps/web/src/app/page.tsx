"use client";

import { AISDashboard } from "@/components/ais-dashboard";
import { GraphBackendStatus } from "@/components/graph-backend-status";

export default function Home() {
  return (
    <div className="container mx-auto max-w-5xl px-4 py-8 space-y-8">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">MAGI Telemetry Portal</h1>
        <p className="text-muted-foreground mt-1">
          Plataforma de monitoreo y consulta de telemetría marítima AIS con autenticación Zitadel.
        </p>
      </div>

      <AISDashboard />

      <section className="rounded-lg border p-4">
        <h2 className="mb-2 font-medium text-sm">Estado de Conexión del Backend</h2>
        <GraphBackendStatus />
      </section>
    </div>
  );
}

