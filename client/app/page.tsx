"use client";

import { Suspense } from "react";
import { useProducts } from "@/hooks/useProducts";
import { ProductGrid } from "@/components/products/ProductGrid";
import { ProductGridSkeleton } from "@/components/ui/product-skeleton";
import { useSearchParams } from "next/navigation";
import { useMemo } from "react";

function ProductsContent() {
  const { data, isLoading, isError } = useProducts();
  const searchParams = useSearchParams();
  const searchQuery = searchParams.get("search") || "";

  const filteredProducts = useMemo(() => {
    if (!data?.products || !searchQuery) {
      return data?.products || [];
    }
    
    const query = searchQuery.toLowerCase();
    return data.products.filter(product => 
      product.name.toLowerCase().includes(query) ||
      product.category.toLowerCase().includes(query) ||
      product.description?.toLowerCase().includes(query)
    );
  }, [data?.products, searchQuery]);

  if (isLoading) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
        <div className="mb-8">
          <h1 className="text-3xl font-bold tracking-tight">Products</h1>
          <p className="text-muted-foreground mt-1">Loading products...</p>
        </div>
        <ProductGridSkeleton />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6 text-center text-destructive">
        Failed to load products. Please try again.
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
      <div className="mb-8">
        <h1 className="text-3xl font-bold tracking-tight">
          {searchQuery ? `Search results for "${searchQuery}"` : "Products"}
        </h1>
        <p className="text-muted-foreground mt-1">
          {searchQuery 
            ? `${filteredProducts.length} of ${data?.total ?? 0} products found`
            : `${data?.total ?? 0} items available`
          }
        </p>
      </div>
      <ProductGrid products={filteredProducts} />
    </div>
  );
}

export default function HomePage() {
  return (
    <Suspense fallback={
      <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
        <div className="mb-8">
          <h1 className="text-3xl font-bold tracking-tight">Products</h1>
          <p className="text-muted-foreground mt-1">Loading...</p>
        </div>
        <ProductGridSkeleton />
      </div>
    }>
      <ProductsContent />
    </Suspense>
  );
}
