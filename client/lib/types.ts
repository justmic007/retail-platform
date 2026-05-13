// ── Auth ──────────────────────────────────────────────────────────────────────

export interface User {
  id: string;
  email: string;
  role: "customer" | "admin";
  email_verified: boolean;
  created_at: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface RegisterRequest {
  email: string;
  password: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

// ── Products ──────────────────────────────────────────────────────────────────

export interface Product {
  id: string;
  sku: string;
  name: string;
  description: string;
  price: number;
  category: string;
  is_active: boolean;
  quantity: number;
  reserved: number;
  available: number;
}

export interface StockLevel {
  product_id: string;
  quantity: number;
  reserved: number;
  available: number;
}

// ── Orders ────────────────────────────────────────────────────────────────────

export type OrderStatus = "PENDING" | "PROCESSING" | "CONFIRMED" | "FAILED" | "CANCELLED";
export type PaymentStatus = "UNPAID" | "PAID" | "REFUNDED";

export interface OrderItem {
  id: string;
  product_id: string;
  product_name: string;
  quantity: number;
  unit_price: string;
  total_price: string;
}

export interface Order {
  id: string;
  user_id: string;
  status: OrderStatus;
  payment_status: PaymentStatus;
  total_amount: string;
  idempotency_key: string;
  notes: string;
  items: OrderItem[];
  created_at: string;
  updated_at: string;
}

export interface CreateOrderRequest {
  idempotency_key: string;
  items: { product_id: string; quantity: number }[];
  notes?: string;
}

// ── Cart ──────────────────────────────────────────────────────────────────────

export interface CartItem {
  product: Product;
  quantity: number;
}

// ── API errors ────────────────────────────────────────────────────────────────

export interface APIError {
  error: string;
  code: string;
}
