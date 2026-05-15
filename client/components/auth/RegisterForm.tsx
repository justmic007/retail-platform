"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { ValidatedInput } from "@/components/ui/validated-input";
import { ValidatedPasswordInput } from "@/components/ui/validated-password-input";
import { register } from "@/lib/api";
import { useFormValidation } from "@/hooks/useFormValidation";

export function RegisterForm() {
  const router = useRouter();
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
        minLength: 8,
        custom: (value: string) => {
          if (!/(?=.*[a-z])/.test(value)) return "Password must contain at least one lowercase letter";
          if (!/(?=.*[A-Z])/.test(value)) return "Password must contain at least one uppercase letter";
          if (!/(?=.*\d)/.test(value)) return "Password must contain at least one number";
          return null;
        }
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
      await register(values.email, values.password);
      toast.success("Account created! Check your email to verify.");
      router.push("/verify-email");
    } catch (err: unknown) {
      const e = err as Error & { code?: string };
      if (e.code === "CONFLICT") {
        setError("An account with this email already exists.");
        toast.error("An account with this email already exists");
      } else {
        setError(e.message ?? "Registration failed. Please try again.");
        toast.error("Registration failed. Please try again.");
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
        placeholder="Min. 8 characters with uppercase, lowercase, and number"
        value={formState.password.value}
        onChange={(e) => setValue("password", e.target.value)}
        onBlur={() => setTouched("password")}
        error={formState.password.error}
        touched={formState.password.touched}
        autoComplete="new-password"
      />
      
      {error && (
        <div className="text-sm text-destructive bg-destructive/10 p-3 rounded flex items-center gap-2">
          <span>{error}</span>
        </div>
      )}
      
      <Button type="submit" className="w-full" disabled={loading}>
        {loading ? "Creating account..." : "Create account"}
      </Button>
    </form>
  );
}
