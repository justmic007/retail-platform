import Cookies from "js-cookie";

const ACCESS_TOKEN_KEY = "access_token";
const REFRESH_TOKEN_KEY = "refresh_token";

export function setTokens(accessToken: string, refreshToken: string): void {
  // Access token — session cookie (expires when browser closes)
  Cookies.set(ACCESS_TOKEN_KEY, accessToken, { sameSite: "strict" });
  // Refresh token — 7 day cookie
  Cookies.set(REFRESH_TOKEN_KEY, refreshToken, { expires: 7, sameSite: "strict" });
}

export function getAccessToken(): string | undefined {
  return Cookies.get(ACCESS_TOKEN_KEY);
}

export function getRefreshToken(): string | undefined {
  return Cookies.get(REFRESH_TOKEN_KEY);
}

export function clearTokens(): void {
  Cookies.remove(ACCESS_TOKEN_KEY);
  Cookies.remove(REFRESH_TOKEN_KEY);
}

export function isLoggedIn(): boolean {
  return !!Cookies.get(ACCESS_TOKEN_KEY);
}
