"use client";

import { useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { ValidatedInput } from "@/components/ui/validated-input";
import { ValidatedPasswordInput } from "@/components/ui/validated-password-input";
import { useAuth } from "@/hooks/useAuth";
import { useFormValidation } from "@/hooks/useFormValidation";

export function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { login } = useAuth();
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const { formState, setValue, setTouched, validateAll, getValues } = useFormValidation(
    { email: "", password: "" },
    {
      email: {
        required: true,
        pattern: /^[^\s@]+@[^\s@]+\.[^\s@]+$/
      },
      password: {
        required: true,
        minLength: 1
      }
    }
  );

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    
    if (!validateAll()) {
      toast.error("Please fix the errors below");
      return;
    }

    setLoading(true);
    try {
      const values = getValues();
      await login(values.email, values.password);
      const redirect = searchParams.get("redirect") ?? "/";
      toast.success("Welcome back!");
      router.push(redirect);
    } catch (err: unknown) {
      const e = err as Error & { code?: string };
      if (e.code === "UNAUTHORIZED") {
        setError("Invalid email or password.");
        toast.error("Invalid email or password");
      } else if (e.message?.includes("verify")) {
        setError("Please verify your email before signing in.");
        toast.error("Please verify your email before signing in");
      } else {
        setError(e.message ?? "Sign in failed. Please try again.");
        toast.error("Sign in failed. Please try again.");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <ValidatedInput
        id="email"
        type="email"
        label="Email"
        placeholder="you@example.com"
        value={formState.email.value}
        onChange={(e) => setValue("email", e.target.value)}
        onBlur={() => setTouched("email")}
        error={formState.email.error}
        touched={formState.email.touched}
        autoComplete="email"
      />
      
      <ValidatedPasswordInput
        id="password"
        label="Password"
        placeholder="Your password"
        value={formState.password.value}
        onChange={(e) => setValue("password", e.target.value)}
        onBlur={() => setTouched("password")}
        error={formState.password.error}
        touched={formState.password.touched}
        autoComplete="current-password"
      />
      
      {error && (
        <div className="text-sm text-destructive bg-destructive/10 p-3 rounded flex items-center gap-2">
          <span>{error}</span>
        </div>
      )}
      
      <Button type="submit" className="w-full" disabled={loading}>
        {loading ? "Signing in..." : "Sign in"}
      </Button>
    </form>
  );
}
