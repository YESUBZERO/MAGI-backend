import { getSession } from "next-auth/react";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export class ApiError extends Error {
  constructor(public status: number, message: string, public data?: unknown) {
    super(message);
    this.name = "ApiError";
  }
}

function isSafeInternalUrl(endpoint: string, base: string): boolean {
  try {
    const baseUrlObj = new URL(base);
    const targetUrlObj = new URL(endpoint, baseUrlObj);
    return targetUrlObj.origin === baseUrlObj.origin;
  } catch {
    return false;
  }
}

/**
 * Cliente HTTP unificado con patrón Token Relay.
 * Inyecta automáticamente el Bearer JWT del usuario autenticado en Zitadel
 * hacia el backend de Go.
 */
export async function apiClient<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const session = await getSession();
  const token = session?.accessToken;

  const headers = new Headers(options.headers);
  headers.set("Content-Type", "application/json");

  // Sanitización P0: Sólo adjuntar Bearer token si el origen coincide estrictamente con API_BASE_URL
  if (token && isSafeInternalUrl(endpoint, API_BASE_URL)) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const url = endpoint.startsWith("http") ? endpoint : `${API_BASE_URL}${endpoint}`;

  const response = await fetch(url, {
    ...options,
    headers,
  });

  if (!response.ok) {
    let errBody: { error?: string } = {};
    try {
      errBody = await response.json();
    } catch {
      // Ignora error si la respuesta no es JSON
    }
    throw new ApiError(
      response.status,
      errBody.error || `HTTP Error ${response.status}: ${response.statusText}`,
      errBody
    );
  }

  return response.json() as Promise<T>;
}
