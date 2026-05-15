"use client";

import { useState } from "react";
import Link from "next/link";
import { ShoppingCart } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { useCartStore } from "@/lib/cart";
import type { Product } from "@/lib/types";

interface Props {
  product: Product;
}

export function ProductCard({ product }: Props) {
  const addItem = useCartStore((s) => s.addItem);
  const [imageError, setImageError] = useState(false);

  const stockBadge =
    product.available === 0
      ? { label: "Out of stock", variant: "destructive" as const }
      : product.available <= 10
      ? { label: `Only ${product.available} left`, variant: "secondary" as const }
      : { label: "In stock", variant: "outline" as const };

  return (
    <Card className="flex flex-col overflow-hidden">
      <Link href={`/products/${product.id}`} className="relative aspect-square bg-muted overflow-hidden">
        {product.image_url && !imageError ? (
          <img
            src={product.image_url}
            alt={product.name}
            className="w-full h-full object-cover transition-transform hover:scale-105"
            onError={() => setImageError(true)}
            loading="lazy"
          />
        ) : (
          <div className="flex h-full items-center justify-center text-muted-foreground text-sm">
            No image
          </div>
        )}
      </Link>

      <CardContent className="flex-1 p-4 space-y-1">
        <Badge variant={stockBadge.variant} className="text-xs">
          {stockBadge.label}
        </Badge>
        <Link href={`/products/${product.id}`}>
          <h3 className="font-medium text-sm leading-snug hover:underline line-clamp-2">
            {product.name}
          </h3>
        </Link>
        <p className="text-xs text-muted-foreground">{product.category}</p>
        <p className="font-semibold">R{product.price.toFixed(2)}</p>
      </CardContent>

      <CardFooter className="p-4 pt-0">
        <Button
          className="w-full"
          size="sm"
          disabled={product.available === 0}
          onClick={() => {
            addItem(product);
            toast.success(`Added ${product.name} to cart`);
          }}
        >
          <ShoppingCart className="h-4 w-4 mr-2" />
          Add to cart
        </Button>
      </CardFooter>
    </Card>
  );
}
