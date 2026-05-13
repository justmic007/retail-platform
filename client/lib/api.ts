import type {
  TokenPair,
  User,
  Product,
  StockLevel,
  Order,
  CreateOrderRequest,
} from "./types";
import { getAccessToken } from "./auth";

const AUTH_URL = process.env.NEXT_PUBLIC_AUTH_URL ?? "http://localhost:8080";
const INVENTORY_URL = process.env.NEXT_PUBLIC_INVENTORY_URL ?? "http://localhost:8082";
const ORDER_URL = process.env.NEXT_PUBLIC_ORDER_URL ?? "http://localhost:8081";

// ── Helpers ───────────────────────────────────────────────────────────────────

async function request<T>(url: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: "Unknown error", code: "UNKNOWN" }));
    const err = new Error(body.error ?? "Request failed") as Error & { code: string; status: number };
    err.code = body.code ?? "UNKNOWN";
    err.status = res.status;
    throw err;
  }

  return res.json();
}

function authHeaders(): Record<string, string> {
  const token = getAccessToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}

// ── Auth ──────────────────────────────────────────────────────────────────────

export async function register(email: string, password: string): Promise<{ user: User }> {
  return request(`${AUTH_URL}/auth/register`, {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
}

export async function login(email: string, password: string): Promise<TokenPair> {
  return request(`${AUTH_URL}/auth/login`, {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
}

export async function logout(refreshToken: string): Promise<void> {
  return request(`${AUTH_URL}/auth/logout`, {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
}

export async function getMe(): Promise<{ user: User }> {
  return request(`${AUTH_URL}/auth/me`, {
    headers: authHeaders(),
  });
}

export async function changePassword(currentPassword: string, newPassword: string): Promise<{ message: string }> {
  return request(`${AUTH_URL}/auth/change-password`, {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
  });
}

// ── Products ──────────────────────────────────────────────────────────────────

export async function getProducts(): Promise<{ products: Product[]; total: number }> {
  return request(`${INVENTORY_URL}/products`);
}

export async function getProduct(id: string): Promise<{ product: Product }> {
  return request(`${INVENTORY_URL}/products/${id}`);
}

export async function getStockLevel(id: string): Promise<StockLevel> {
  return request(`${INVENTORY_URL}/products/${id}/stock`);
}

// ── Orders ────────────────────────────────────────────────────────────────────

export async function createOrder(data: CreateOrderRequest): Promise<{ message: string; order: Order }> {
  return request(`${ORDER_URL}/orders`, {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify(data),
  });
}

export async function getOrders(): Promise<{ orders: Order[]; total: number }> {
  return request(`${ORDER_URL}/orders`, {
    headers: authHeaders(),
  });
}

export async function getOrder(id: string): Promise<{ order: Order }> {
  return request(`${ORDER_URL}/orders/${id}`, {
    headers: authHeaders(),
  });
}

export async function cancelOrder(id: string): Promise<{ message: string }> {
  return request(`${ORDER_URL}/orders/${id}/cancel`, {
    method: "PATCH",
    headers: authHeaders(),
  });
}
