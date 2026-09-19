import { createEnv } from "@t3-oss/env-nextjs";
import { z } from "zod";

export const env = createEnv({
  server: {
    NEXTAUTH_SECRET: z.string().min(1).optional(),
    NEXTAUTH_URL: z.string().url().optional(),
    ZITADEL_ISSUER: z.string().url().optional(),
    ZITADEL_CLIENT_ID: z.string().min(1).optional(),
    ZITADEL_CLIENT_SECRET: z.string().min(1).optional(),
    TURNSTILE_SECRET_KEY: z.string().min(1).optional(),
  },
  client: {
    NEXT_PUBLIC_API_URL: z.string().url().optional().default("http://localhost:8080"),
    NEXT_PUBLIC_TURNSTILE_SITE_KEY: z.string().min(1).optional(),
  },
  runtimeEnv: {
    NEXTAUTH_SECRET: process.env.NEXTAUTH_SECRET,
    NEXTAUTH_URL: process.env.NEXTAUTH_URL,
    ZITADEL_ISSUER: process.env.ZITADEL_ISSUER,
    ZITADEL_CLIENT_ID: process.env.ZITADEL_CLIENT_ID,
    ZITADEL_CLIENT_SECRET: process.env.ZITADEL_CLIENT_SECRET,
    TURNSTILE_SECRET_KEY: process.env.TURNSTILE_SECRET_KEY,
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL,
    NEXT_PUBLIC_TURNSTILE_SITE_KEY: process.env.NEXT_PUBLIC_TURNSTILE_SITE_KEY,
  },
  emptyStringAsUndefined: true,
});

