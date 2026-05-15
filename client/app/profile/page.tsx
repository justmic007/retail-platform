"use client";

import { useState } from "react";
import { User, Lock, Mail, Calendar } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ValidatedPasswordInput } from "@/components/ui/validated-password-input";
import { useAuth } from "@/hooks/useAuth";
import { changePassword } from "@/lib/api";
import { useFormValidation } from "@/hooks/useFormValidation";

export default function ProfilePage() {
  const { user, loading: authLoading } = useAuth();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const { formState, setValue, setTouched, validateAll, getValues, reset } = useFormValidation(
    { currentPassword: "", newPassword: "", confirmPassword: "" },
    {
      currentPassword: {
        required: true,
        minLength: 1
      },
      newPassword: {
        required: true,
        minLength: 8,
        custom: (value: string) => {
          if (!/(?=.*[a-z])/.test(value)) return "Password must contain at least one lowercase letter";
          if (!/(?=.*[A-Z])/.test(value)) return "Password must contain at least one uppercase letter";
          if (!/(?=.*\d)/.test(value)) return "Password must contain at least one number";
          return null;
        }
      },
      confirmPassword: {
        required: true,
        custom: (value: string) => {
          const newPassword = formState.newPassword?.value || "";
          return value !== newPassword ? "Passwords do not match" : null;
        }
      }
    }
  );

  if (authLoading) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-12 sm:px-6">
        <div className="animate-pulse space-y-6">
          <div className="h-8 bg-muted rounded w-1/4" />
          <div className="h-64 bg-muted rounded" />
        </div>
      </div>
    );
  }

  if (!user) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-16 sm:px-6 text-center">
        <h1 className="text-2xl font-bold mb-4">Please sign in</h1>
        <p className="text-muted-foreground">You need to be logged in to view your profile.</p>
      </div>
    );
  }

  async function handleChangePassword(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setSuccess("");

    if (!validateAll()) {
      toast.error("Please fix the errors below");
      return;
    }

    setLoading(true);
    try {
      const values = getValues();
      await changePassword(values.currentPassword, values.newPassword);
      setSuccess("Password changed successfully. Please log in again on all devices.");
      toast.success("Password changed successfully!");
      reset();
    } catch (err: unknown) {
      const e = err as Error & { code?: string };
      if (e.code === "UNAUTHORIZED") {
        setError("Current password is incorrect");
        toast.error("Current password is incorrect");
      } else {
        setError(e.message || "Failed to change password");
        toast.error("Failed to change password");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="mx-auto max-w-2xl px-4 py-8 sm:px-6">
      <h1 className="text-2xl font-bold mb-6">Profile</h1>

      <div className="space-y-6">
        {/* Account Information */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <User className="h-5 w-5" />
              Account Information
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center gap-3">
              <Mail className="h-4 w-4 text-muted-foreground" />
              <div>
                <p className="text-sm text-muted-foreground">Email</p>
                <p className="font-medium">{user.email}</p>
              </div>
            </div>
            
            <div className="flex items-center gap-3">
              <User className="h-4 w-4 text-muted-foreground" />
              <div>
                <p className="text-sm text-muted-foreground">Role</p>
                <p className="font-medium capitalize">{user.role}</p>
              </div>
            </div>

            <div className="flex items-center gap-3">
              <Calendar className="h-4 w-4 text-muted-foreground" />
              <div>
                <p className="text-sm text-muted-foreground">Member since</p>
                <p className="font-medium">
                  {new Date(user.created_at).toLocaleDateString()}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Change Password */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Lock className="h-5 w-5" />
              Change Password
            </CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleChangePassword} className="space-y-4">
              <ValidatedPasswordInput
                id="current-password"
                label="Current Password"
                value={formState.currentPassword.value}
                onChange={(e) => setValue("currentPassword", e.target.value)}
                onBlur={() => setTouched("currentPassword")}
                error={formState.currentPassword.error}
                touched={formState.currentPassword.touched}
                autoComplete="current-password"
                showValidIcon={false}
              />

              <ValidatedPasswordInput
                id="new-password"
                label="New Password"
                placeholder="Min. 8 characters with uppercase, lowercase, and number"
                value={formState.newPassword.value}
                onChange={(e) => {
                  setValue("newPassword", e.target.value);
                  // Re-validate confirm password when new password changes
                  if (formState.confirmPassword.touched) {
                    setTouched("confirmPassword");
                  }
                }}
                onBlur={() => setTouched("newPassword")}
                error={formState.newPassword.error}
                touched={formState.newPassword.touched}
                autoComplete="new-password"
              />

              <ValidatedPasswordInput
                id="confirm-password"
                label="Confirm New Password"
                value={formState.confirmPassword.value}
                onChange={(e) => setValue("confirmPassword", e.target.value)}
                onBlur={() => setTouched("confirmPassword")}
                error={formState.confirmPassword.error}
                touched={formState.confirmPassword.touched}
                autoComplete="new-password"
              />

              {error && (
                <div className="text-sm text-destructive bg-destructive/10 p-3 rounded">
                  {error}
                </div>
              )}

              {success && (
                <div className="text-sm text-green-700 bg-green-50 p-3 rounded">
                  {success}
                </div>
              )}

              <Button type="submit" disabled={loading}>
                {loading ? "Changing Password..." : "Change Password"}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}