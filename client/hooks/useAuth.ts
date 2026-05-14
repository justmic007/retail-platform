"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { getMe, login as apiLogin, logout as apiLogout } from "@/lib/api";
import { setTokens, clearTokens, getRefreshToken, isLoggedIn } from "@/lib/auth";
import type { User } from "@/lib/types";

export function useAuth() {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  useEffect(() => {
    if (!isLoggedIn()) {
      setLoading(false);
      return;
    }
    getMe()
      .then(({ user }) => setUser(user))
      .catch(() => clearTokens())
      .finally(() => setLoading(false));
  }, []);

  async function login(email: string, password: string) {
    const tokens = await apiLogin(email, password);
    setTokens(tokens.access_token, tokens.refresh_token);
    const { user } = await getMe();
    setUser(user);
    return user;
  }

  async function logout() {
    const refreshToken = getRefreshToken();
    if (refreshToken) {
      await apiLogout(refreshToken).catch(() => {});
    }
    clearTokens();
    setUser(null);
    router.push("/");
  }

  return { user, loading, login, logout };
}
