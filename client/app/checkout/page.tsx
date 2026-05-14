"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { ArrowLeft, CreditCard } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useCartStore } from "@/lib/cart";
import { useAuth } from "@/hooks/useAuth";
import { createOrder } from "@/lib/api";

export default function CheckoutPage() {
  const router = useRouter();
  const { user, loading: authLoading } = useAuth();
  const { items, totalPrice, clearCart } = useCartStore();
  const [notes, setNotes] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [isPlacingOrder, setIsPlacingOrder] = useState(false);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (!mounted || authLoading) return;
    
    if (!user) {
      router.push("/login?redirect=/checkout");
      return;
    }

    if (items.length === 0 && !isPlacingOrder) {
      router.push("/cart");
      return;
    }
  }, [mounted, authLoading, user, items.length, router, isPlacingOrder]);

  if (!mounted || authLoading) {
    return (
      <div className="mx-auto max-w-4xl px-4 py-8 sm:px-6">
        <div className="animate-pulse space-y-6">
          <div className="h-8 bg-muted rounded w-1/4" />
          <div className="h-64 bg-muted rounded" />
        </div>
      </div>
    );
  }

  if (!user || items.length === 0) {
    return null;
  }

  async function handlePlaceOrder() {
    setLoading(true);
    setError("");
    setIsPlacingOrder(true);
    
    try {
      const idempotencyKey = `checkout-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
      
      const orderData = {
        idempotency_key: idempotencyKey,
        items: items.map(item => ({
          product_id: item.product.id,
          quantity: item.quantity
        })),
        notes: notes.trim() || undefined
      };

      const response = await createOrder(orderData);
      
      // Navigate to order confirmation first, then clear cart
      router.push(`/orders/${response.order.id}`);
      
      // Clear cart after navigation to prevent useEffect redirect
      setTimeout(() => {
        clearCart();
        setIsPlacingOrder(false);
      }, 100);
      
    } catch (err: unknown) {
      const e = err as Error & { code?: string };
      if (e.code === "INSUFFICIENT_STOCK") {
        setError("Some items are no longer available. Please check your cart.");
      } else {
        setError(e.message || "Failed to place order. Please try again.");
      }
      setIsPlacingOrder(false);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="mx-auto max-w-4xl px-4 py-8 sm:px-6">
      <Button variant="ghost" size="sm" className="mb-6" onClick={() => router.back()}>
        <ArrowLeft className="h-4 w-4 mr-2" />
        Back to Cart
      </Button>

      <h1 className="text-2xl font-bold mb-6">Checkout</h1>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Order Review */}
        <div className="lg:col-span-2 space-y-6">
          {/* Customer Info */}
          <Card>
            <CardHeader>
              <CardTitle>Customer Information</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">Logged in as:</p>
              <p className="font-medium">{user.email}</p>
            </CardContent>
          </Card>

          {/* Order Items */}
          <Card>
            <CardHeader>
              <CardTitle>Order Items</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {items.map((item) => (
                <div key={item.product.id} className="flex justify-between items-center py-2 border-b last:border-b-0">
                  <div className="flex-1">
                    <h4 className="font-medium text-sm">{item.product.name}</h4>
                    <p className="text-xs text-muted-foreground">
                      R{item.product.price.toFixed(2)} × {item.quantity}
                    </p>
                  </div>
                  <div className="text-sm font-medium">
                    R{(item.product.price * item.quantity).toFixed(2)}
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>

          {/* Order Notes */}
          <Card>
            <CardHeader>
              <CardTitle>Order Notes (Optional)</CardTitle>
            </CardHeader>
            <CardContent>
              <Input
                placeholder="Special instructions for your order..."
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                maxLength={500}
              />
              <p className="text-xs text-muted-foreground mt-1">
                {notes.length}/500 characters
              </p>
            </CardContent>
          </Card>
        </div>

        {/* Order Summary & Payment */}
        <div className="lg:col-span-1">
          <Card>
            <CardHeader>
              <CardTitle>Order Summary</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex justify-between text-sm">
                <span>Subtotal</span>
                <span>R{totalPrice().toFixed(2)}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span>Shipping</span>
                <span>Free</span>
              </div>
              <div className="flex justify-between text-sm">
                <span>Tax</span>
                <span>Included</span>
              </div>
              <hr />
              <div className="flex justify-between font-semibold text-lg">
                <span>Total</span>
                <span>R{totalPrice().toFixed(2)}</span>
              </div>

              {error && (
                <div className="text-sm text-destructive bg-destructive/10 p-3 rounded">
                  {error}
                </div>
              )}

              <Button 
                className="w-full" 
                size="lg"
                onClick={handlePlaceOrder}
                disabled={loading}
              >
                <CreditCard className="h-4 w-4 mr-2" />
                {loading ? "Placing Order..." : "Place Order"}
              </Button>

              <p className="text-xs text-muted-foreground text-center">
                By placing this order, you agree to our terms and conditions.
                Payment will be processed upon delivery.
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}