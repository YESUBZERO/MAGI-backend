import type { NextAuthOptions } from "next-auth";
import ZitadelProvider from "next-auth/providers/zitadel";
import { env } from "@aires-magi/env/web";

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
    async jwt({ token, account }) {
      if (account) {
        token.accessToken = account.access_token;
        token.idToken = account.id_token;
      }
      return token;
    },
    async session({ session, token }) {
      if (session.user) {
        (session as unknown as { accessToken?: string }).accessToken = token.accessToken as string | undefined;
      }
      return session;
    },
  },
};
