"use client";

import { use, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { ArrowLeft, Package, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { getOrder, cancelOrder } from "@/lib/api";
import { useAuth } from "@/hooks/useAuth";
import type { Order } from "@/lib/types";

interface Props {
  params: Promise<{ id: string }>;
}

function getStatusBadge(status: Order["status"]) {
  switch (status) {
    case "PENDING":
      return <Badge variant="secondary">Processing</Badge>;
    case "PROCESSING":
      return <Badge variant="secondary">Processing</Badge>;
    case "CONFIRMED":
      return <Badge variant="default">Confirmed</Badge>;
    case "FAILED":
      return <Badge variant="destructive">Failed</Badge>;
    case "CANCELLED":
      return <Badge variant="outline">Cancelled</Badge>;
    default:
      return <Badge variant="outline">{status}</Badge>;
  }
}

export default function OrderDetailPage({ params }: Props) {
  const { id } = use(params);
  const router = useRouter();
  const { user } = useAuth();
  const [cancelling, setCancelling] = useState(false);

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ["orders", id],
    queryFn: () => getOrder(id),
    enabled: !!user && !!id,
  });

  if (!user) {
    router.push("/login?redirect=/orders");
    return null;
  }

  if (isLoading) {
    return (
      <div className="mx-auto max-w-4xl px-4 py-12 sm:px-6">
        <div className="animate-pulse space-y-6">
          <div className="h-8 bg-muted rounded w-1/4" />
          <div className="h-64 bg-muted rounded" />
        </div>
      </div>
    );
  }

  if (isError || !data?.order) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-16 sm:px-6 text-center">
        <h1 className="text-2xl font-bold mb-4">Order not found</h1>
        <p className="text-muted-foreground mb-6">This order doesn't exist or you don't have permission to view it.</p>
        <Button asChild>
          <ArrowLeft className="h-4 w-4 mr-2" />
          <span>Back to Orders</span>
        </Button>
      </div>
    );
  }

  const order = data.order;
  const canCancel = order.status === "PENDING";

  async function handleCancel() {
    if (!canCancel) return;

    setCancelling(true);
    try {
      await cancelOrder(order.id);
      refetch(); // Refresh order data
    } catch (error) {
      console.error("Failed to cancel order:", error);
    } finally {
      setCancelling(false);
    }
  }

  return (
    <div className="mx-auto max-w-4xl px-4 py-8 sm:px-6">
      <Button variant="ghost" size="sm" className="mb-6" onClick={() => router.push("/orders")}>
        <ArrowLeft className="h-4 w-4 mr-2" />
        Back to Orders
      </Button>

      <div className="space-y-6">
        {/* Order Header */}
        <Card>
          <CardHeader>
            <div className="flex items-start justify-between">
              <div>
                <CardTitle className="flex items-center gap-3">
                  <Package className="h-5 w-5" />
                  Order #{order.id.slice(-8)}
                  {getStatusBadge(order.status)}
                </CardTitle>
                <p className="text-sm text-muted-foreground mt-1">
                  Placed on {new Date(order.created_at).toLocaleDateString()} at{" "}
                  {new Date(order.created_at).toLocaleTimeString()}
                </p>
              </div>

              {canCancel && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleCancel}
                  disabled={cancelling}
                  className="text-destructive hover:text-destructive"
                >
                  <X className="h-4 w-4 mr-2" />
                  {cancelling ? "Cancelling..." : "Cancel Order"}
                </Button>
              )}
            </div>
          </CardHeader>
        </Card>

        {/* Order Items */}
        <Card>
          <CardHeader>
            <CardTitle>Order Items</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {order.items.map((item) => (
                <div key={item.id} className="flex justify-between items-start py-3 border-b last:border-b-0">
                  <div className="flex-1">
                    <h4 className="font-medium">{item.product_name}</h4>
                    <p className="text-sm text-muted-foreground">
                      Quantity: {item.quantity}
                    </p>
                    <p className="text-sm text-muted-foreground">
                      Unit price: ${parseFloat(item.unit_price).toFixed(2)}
                    </p>
                  </div>
                  <div className="text-right">
                    <p className="font-medium">${parseFloat(item.total_price).toFixed(2)}</p>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Order Summary */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <Card>
            <CardHeader>
              <CardTitle>Order Details</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              <div className="flex justify-between text-sm">
                <span>Order ID:</span>
                <span className="font-mono text-xs">{order.id}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span>Status:</span>
                <span>{getStatusBadge(order.status)}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span>Payment Status:</span>
                <span className="capitalize">{order.payment_status.toLowerCase()}</span>
              </div>
              {order.notes && (
                <div className="pt-2">
                  <p className="text-sm font-medium">Notes:</p>
                  <p className="text-sm text-muted-foreground">{order.notes}</p>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Order Total</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              <div className="flex justify-between text-sm">
                <span>Subtotal:</span>
                <span>${parseFloat(order.total_amount).toFixed(2)}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span>Shipping:</span>
                <span>Free</span>
              </div>
              <div className="flex justify-between text-sm">
                <span>Tax:</span>
                <span>Included</span>
              </div>
              <hr />
              <div className="flex justify-between font-semibold">
                <span>Total:</span>
                <span>R{parseFloat(order.total_amount).toFixed(2)}</span>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}