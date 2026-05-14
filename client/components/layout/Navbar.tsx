"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ShoppingCart, User, Search, ChevronDown, Menu, X } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuth } from "@/hooks/useAuth";
import { useCartStore } from "@/lib/cart";

export function Navbar() {
  const { user, logout } = useAuth();
  const totalItems = useCartStore((s) => s.totalItems);
  const router = useRouter();
  const [isHydrated, setIsHydrated] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [showAccountMenu, setShowAccountMenu] = useState(false);
  const [showMobileMenu, setShowMobileMenu] = useState(false);

  useEffect(() => {
    setIsHydrated(true);
  }, []);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (searchQuery.trim()) {
      router.push(`/?search=${encodeURIComponent(searchQuery.trim())}`);
      setShowMobileMenu(false);
    }
  };

  const closeMenus = () => {
    setShowAccountMenu(false);
    setShowMobileMenu(false);
  };

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur">
      <div className="mx-auto max-w-7xl px-4 sm:px-6">
        {/* Desktop Layout */}
        <div className="hidden md:flex h-16 items-center justify-between gap-4">
          {/* Logo */}
          <Link href="/" className="text-xl font-bold tracking-tight flex-shrink-0">
            RetailPlatform
          </Link>

          {/* Search Bar */}
          <form onSubmit={handleSearch} className="flex-1 max-w-md mx-4">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                type="search"
                placeholder="Search products..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10 pr-4"
              />
            </div>
          </form>

          {/* Right side actions */}
          <div className="flex items-center gap-2">
            {/* Account Menu */}
            <div className="relative">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowAccountMenu(!showAccountMenu)}
                className="flex items-center gap-1"
              >
                <User className="h-4 w-4" />
                <span className="hidden lg:inline">
                  {user ? "My Account" : "Account"}
                </span>
                <ChevronDown className="h-3 w-3" />
              </Button>
              
              {showAccountMenu && (
                <div className="absolute right-0 top-full mt-1 w-48 rounded-md border bg-background shadow-lg z-50">
                  <div className="py-1">
                    {user ? (
                      <>
                        <Link
                          href="/profile"
                          className="block px-4 py-2 text-sm hover:bg-muted"
                          onClick={closeMenus}
                        >
                          My Account
                        </Link>
                        <Link
                          href="/orders"
                          className="block px-4 py-2 text-sm hover:bg-muted"
                          onClick={closeMenus}
                        >
                          Orders
                        </Link>
                        <div className="border-t my-1" />
                        <button
                          onClick={() => {
                            logout();
                            closeMenus();
                            toast.success("Signed out successfully");
                          }}
                          className="block w-full text-left px-4 py-2 text-sm hover:bg-muted"
                        >
                          Sign Out
                        </button>
                      </>
                    ) : (
                      <>
                        <Link
                          href="/login"
                          className="block px-4 py-2 text-sm hover:bg-muted"
                          onClick={closeMenus}
                        >
                          Sign In
                        </Link>
                        <Link
                          href="/register"
                          className="block px-4 py-2 text-sm hover:bg-muted"
                          onClick={closeMenus}
                        >
                          Create Account
                        </Link>
                      </>
                    )}
                  </div>
                </div>
              )}
            </div>

            {/* Cart */}
            <Button variant="ghost" size="icon" asChild className="relative">
              <Link href="/cart">
                <ShoppingCart className="h-5 w-5" />
                {isHydrated && totalItems() > 0 && (
                  <span className="absolute -top-1 -right-1 flex h-5 w-5 items-center justify-center rounded-full bg-primary text-[10px] font-bold text-primary-foreground">
                    {totalItems() > 99 ? "99+" : totalItems()}
                  </span>
                )}
              </Link>
            </Button>
          </div>
        </div>

        {/* Mobile Layout */}
        <div className="md:hidden">
          {/* Top row - Logo and Menu button */}
          <div className="flex h-16 items-center justify-between">
            <Link href="/" className="text-lg font-bold tracking-tight" onClick={closeMenus}>
              RetailPlatform
            </Link>
            
            <div className="flex items-center gap-2">
              {/* Cart */}
              <Button variant="ghost" size="icon" asChild className="relative">
                <Link href="/cart" onClick={closeMenus}>
                  <ShoppingCart className="h-5 w-5" />
                  {isHydrated && totalItems() > 0 && (
                    <span className="absolute -top-1 -right-1 flex h-5 w-5 items-center justify-center rounded-full bg-primary text-[10px] font-bold text-primary-foreground">
                      {totalItems() > 99 ? "99+" : totalItems()}
                    </span>
                  )}
                </Link>
              </Button>
              
              {/* Mobile menu button */}
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setShowMobileMenu(!showMobileMenu)}
              >
                {showMobileMenu ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
              </Button>
            </div>
          </div>

          {/* Mobile menu */}
          {showMobileMenu && (
            <div className="border-t bg-background">
              <div className="px-4 py-4 space-y-4">
                {/* Search */}
                <form onSubmit={handleSearch}>
                  <div className="relative">
                    <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                    <Input
                      type="search"
                      placeholder="Search products..."
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      className="pl-10 pr-4"
                    />
                  </div>
                </form>

                {/* Account links */}
                <div className="space-y-2">
                  {user ? (
                    <>
                      <div className="text-sm text-muted-foreground px-3 py-2">
                        Signed in as {user.email}
                      </div>
                      <Link
                        href="/profile"
                        className="block px-3 py-2 text-sm hover:bg-muted rounded-md"
                        onClick={closeMenus}
                      >
                        My Account
                      </Link>
                      <Link
                        href="/orders"
                        className="block px-3 py-2 text-sm hover:bg-muted rounded-md"
                        onClick={closeMenus}
                      >
                        Orders
                      </Link>
                      <button
                        onClick={() => {
                          logout();
                          closeMenus();
                          toast.success("Signed out successfully");
                        }}
                        className="block w-full text-left px-3 py-2 text-sm hover:bg-muted rounded-md"
                      >
                        Sign Out
                      </button>
                    </>
                  ) : (
                    <>
                      <Link
                        href="/login"
                        className="block px-3 py-2 text-sm hover:bg-muted rounded-md"
                        onClick={closeMenus}
                      >
                        Sign In
                      </Link>
                      <Link
                        href="/register"
                        className="block px-3 py-2 text-sm hover:bg-muted rounded-md"
                        onClick={closeMenus}
                      >
                        Create Account
                      </Link>
                    </>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Click outside to close menus */}
      {(showAccountMenu || showMobileMenu) && (
        <div
          className="fixed inset-0 z-40"
          onClick={closeMenus}
        />
      )}
    </header>
  );
}
