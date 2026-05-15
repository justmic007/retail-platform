export function ProductSkeleton() {
  return (
    <div className="flex flex-col overflow-hidden rounded-lg border bg-card">
      {/* Image skeleton */}
      <div className="aspect-square bg-muted animate-pulse" />
      
      {/* Content skeleton */}
      <div className="flex-1 p-4 space-y-3">
        {/* Badge skeleton */}
        <div className="h-5 w-20 bg-muted animate-pulse rounded-full" />
        
        {/* Title skeleton */}
        <div className="space-y-2">
          <div className="h-4 bg-muted animate-pulse rounded w-3/4" />
          <div className="h-4 bg-muted animate-pulse rounded w-1/2" />
        </div>
        
        {/* Category skeleton */}
        <div className="h-3 bg-muted animate-pulse rounded w-1/3" />
        
        {/* Price skeleton */}
        <div className="h-5 bg-muted animate-pulse rounded w-1/4" />
      </div>
      
      {/* Button skeleton */}
      <div className="p-4 pt-0">
        <div className="h-9 bg-muted animate-pulse rounded w-full" />
      </div>
    </div>
  );
}

export function ProductGridSkeleton({ count = 8 }: { count?: number }) {
  return (
    <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      {Array.from({ length: count }).map((_, i) => (
        <ProductSkeleton key={i} />
      ))}
    </div>
  );
}