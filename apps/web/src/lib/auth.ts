import type { NextAuthOptions } from "next-auth";
import ZitadelProvider from "next-auth/providers/zitadel";
import { env } from "@aires-magi/env/web";

// Augmentación de tipos para NextAuth y JWT con Token Relay y Roles de Zitadel
declare module "next-auth" {
  interface Session {
    accessToken?: string;
    idToken?: string;
    roles?: string[];
    user?: {
      name?: string | null;
      email?: string | null;
      image?: string | null;
      roles?: string[];
    };
  }
}

declare module "next-auth/jwt" {
  interface JWT {
    accessToken?: string;
    idToken?: string;
    roles?: string[];
  }
}

export const authOptions: NextAuthOptions = {
  providers: [
    ZitadelProvider({
      issuer: env.ZITADEL_ISSUER || "https://issuer.zitadel.ch",
      clientId: env.ZITADEL_CLIENT_ID || "zitadel-client-id",
      clientSecret: env.ZITADEL_CLIENT_SECRET || "",
      authorization: {
        params: {
          scope: "openid profile email urn:zitadel:iam:org:project:id:zitadel:aud",
        },
      },
    }),
  ],
  session: {
    strategy: "jwt",
  },
  callbacks: {
    async jwt({ token, account, profile }) {
      if (account) {
        token.accessToken = account.access_token;
        token.idToken = account.id_token;
      }
      if (profile) {
        // Extracción de roles asignados en el proyecto Zitadel
        const zitadelRoles = (profile as Record<string, unknown>)["urn:zitadel:iam:org:project:roles"];
        if (zitadelRoles && typeof zitadelRoles === "object") {
          token.roles = Object.keys(zitadelRoles);
        } else if (Array.isArray(zitadelRoles)) {
          token.roles = zitadelRoles as string[];
        }
      }
      return token;
    },
    async session({ session, token }) {
      session.accessToken = token.accessToken;
      session.idToken = token.idToken;
      session.roles = token.roles || [];
      if (session.user) {
        session.user.roles = token.roles || [];
      }
      return session;
    },
  },
};
