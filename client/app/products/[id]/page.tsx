"use client";

import { use, useState } from "react";
import { useRouter } from "next/navigation";
import { ShoppingCart, ArrowLeft } from "lucide-react";
import { useProduct } from "@/hooks/useProducts";
import { useCartStore } from "@/lib/cart";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

interface Props {
  params: Promise<{ id: string }>;
}

export default function ProductDetailPage({ params }: Props) {
  const { id } = use(params);
  const router = useRouter();
  const { data, isLoading, isError } = useProduct(id);
  const addItem = useCartStore((s) => s.addItem);
  const [imageError, setImageError] = useState(false);

  if (isLoading) {
    return (
      <div className="mx-auto max-w-5xl px-4 py-12 sm:px-6">
        <div className="animate-pulse">
          {/* Back button skeleton */}
          <div className="h-9 w-20 bg-muted rounded mb-6" />
          
          <div className="grid grid-cols-1 gap-10 md:grid-cols-2">
            {/* Image skeleton */}
            <div className="aspect-square rounded-lg bg-muted" />
            
            {/* Content skeleton */}
            <div className="space-y-4">
              <div className="space-y-2">
                <div className="h-4 bg-muted rounded w-1/3" />
                <div className="h-8 bg-muted rounded w-3/4" />
                <div className="h-4 bg-muted rounded w-1/4" />
              </div>
              <div className="h-10 bg-muted rounded w-1/3" />
              <div className="h-6 bg-muted rounded w-1/4" />
              <div className="space-y-2">
                <div className="h-4 bg-muted rounded w-full" />
                <div className="h-4 bg-muted rounded w-2/3" />
              </div>
              <div className="h-11 bg-muted rounded w-full" />
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (isError || !data?.product) {
    return (
      <div className="mx-auto max-w-5xl px-4 py-12 sm:px-6 text-center text-destructive">
        Product not found.
      </div>
    );
  }

  const product = data.product;
  const outOfStock = product.available === 0;
  const lowStock = product.available > 0 && product.available <= 10;

  return (
    <div className="mx-auto max-w-5xl px-4 py-12 sm:px-6">
      <Button variant="ghost" size="sm" className="mb-6" onClick={() => router.back()}>
        <ArrowLeft className="h-4 w-4 mr-2" />
        Back
      </Button>

      <div className="grid grid-cols-1 gap-10 md:grid-cols-2">
        <div className="relative aspect-square rounded-lg overflow-hidden bg-muted">
          {product.image_url && !imageError ? (
            <img
              src={product.image_url}
              alt={product.name}
              className="w-full h-full object-cover"
              loading="lazy"
              onError={() => setImageError(true)}
            />
          ) : (
            <div className="flex h-full items-center justify-center text-muted-foreground">
              No image available
            </div>
          )}
        </div>

        <div className="space-y-4">
          <div>
            <p className="text-sm text-muted-foreground">{product.category}</p>
            <h1 className="text-2xl font-bold mt-1">{product.name}</h1>
            <p className="text-sm text-muted-foreground mt-1">SKU: {product.sku}</p>
          </div>

          <p className="text-3xl font-bold">${product.price.toFixed(2)}</p>

          {outOfStock ? (
            <Badge variant="destructive">Out of stock</Badge>
          ) : lowStock ? (
            <Badge variant="secondary">Only {product.available} left</Badge>
          ) : (
            <Badge variant="outline">In stock — {product.available} available</Badge>
          )}

          {product.description && (
            <p className="text-muted-foreground text-sm leading-relaxed">{product.description}</p>
          )}

          <Button
            size="lg"
            className="w-full"
            disabled={outOfStock}
            onClick={() => {
              addItem(product);
              router.push("/cart");
            }}
          >
            <ShoppingCart className="h-5 w-5 mr-2" />
            {outOfStock ? "Out of stock" : "Add to cart"}
          </Button>
        </div>
      </div>
    </div>
  );
}
