import NextAuth from "next-auth";
import Google from "next-auth/providers/google";

export const { handlers, auth, signIn, signOut } = NextAuth({
  providers: [Google],
  callbacks: {
    async signIn({ account, profile }) {
      if (account?.provider !== "google") return false;

      const gProfile = profile as { email_verified?: boolean; email?: string; name?: string } | undefined;
      if (!gProfile?.email_verified) return false;

      const email = gProfile.email?.toLowerCase();
      if (!email) return false;

      try {
        const res = await fetch(`${(process.env.API_BASE_URL ?? "").replace(/\/+$/, "")}/v1/me`, {
          headers: {
            "X-Internal-Key": process.env.INTERNAL_API_KEY ?? "",
            "X-User-Email": email,
            "X-User-Name": gProfile.name ?? "",
          },
          cache: "no-store",
        });
        if (!res.ok) return "/denied";
        return true;
      } catch {
        return false;
      }
    },
    async session({ session, token }) {
      if (session.user && token.email) {
        session.user.email = token.email;
      }
      return session;
    },
    async jwt({ token, profile }) {
      const gProfile = profile as { email?: string } | undefined;
      if (gProfile?.email) token.email = gProfile.email.toLowerCase();
      return token;
    },
  },
  pages: {
    signIn: "/login",
    error: "/denied",
  },
  session: { strategy: "jwt", maxAge: 30 * 24 * 60 * 60 },
});
