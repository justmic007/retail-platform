"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ShoppingCart, User, LogOut, Package } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/hooks/useAuth";
import { useCartStore } from "@/lib/cart";

export function Navbar() {
  const { user, logout } = useAuth();
  const totalItems = useCartStore((s) => s.totalItems());
  const router = useRouter();
  const [isHydrated, setIsHydrated] = useState(false);

  useEffect(() => {
    setIsHydrated(true);
  }, []);

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6">
        {/* Logo */}
        <Link href="/" className="text-xl font-bold tracking-tight">
          RetailPlatform
        </Link>

        {/* Nav links */}
        <nav className="hidden sm:flex items-center gap-6 text-sm font-medium">
          <Link href="/" className="text-muted-foreground hover:text-foreground transition-colors">
            Products
          </Link>
          {user && (
            <Link href="/orders" className="text-muted-foreground hover:text-foreground transition-colors">
              Orders
            </Link>
          )}
        </nav>

        {/* Right side */}
        <div className="flex items-center gap-2">
          {/* Cart */}
          <Button variant="ghost" size="icon" asChild className="relative">
            <Link href="/cart">
              <ShoppingCart className="h-5 w-5" />
              {isHydrated && totalItems > 0 && (
                <span className="absolute -top-1 -right-1 flex h-5 w-5 items-center justify-center rounded-full bg-primary text-[10px] font-bold text-primary-foreground">
                  {totalItems > 99 ? "99+" : totalItems}
                </span>
              )}
            </Link>
          </Button>

          {user ? (
            <>
              <Button variant="ghost" size="icon" asChild>
                <Link href="/profile">
                  <User className="h-5 w-5" />
                </Link>
              </Button>
              <Button variant="ghost" size="icon" asChild className="sm:hidden">
                <Link href="/orders">
                  <Package className="h-5 w-5" />
                </Link>
              </Button>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => {
                  logout();
                  toast.success("Signed out successfully");
                }}
                title="Sign out"
              >
                <LogOut className="h-5 w-5" />
              </Button>
            </>
          ) : (
            <>
              <Button variant="ghost" size="sm" onClick={() => router.push("/login")}>
                Sign in
              </Button>
              <Button size="sm" onClick={() => router.push("/register")}>
                Register
              </Button>
            </>
          )}
        </div>
      </div>
    </header>
  );
}
