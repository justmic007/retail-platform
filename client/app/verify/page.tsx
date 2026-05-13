"use client";

import { useEffect, useState } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import Link from "next/link";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

export default function VerifyPage() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const token = searchParams.get("token");
  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");

  useEffect(() => {
    if (!token) {
      setStatus("error");
      return;
    }

    const AUTH_URL = process.env.NEXT_PUBLIC_AUTH_URL ?? "http://localhost:8080";

    fetch(`${AUTH_URL}/auth/verify?token=${token}`)
      .then((res) => {
        if (res.ok) {
          setStatus("success");
          setTimeout(() => router.push("/login"), 2000);
        } else {
          setStatus("error");
        }
      })
      .catch(() => setStatus("error"));
  }, [token, router]);

  return (
    <div className="min-h-screen flex items-center justify-center bg-muted/40 px-4">
      <Card className="w-full max-w-md text-center">
        <CardHeader>
          {status === "loading" && (
            <>
              <div className="mx-auto mb-4 text-4xl">⏳</div>
              <CardTitle>Verifying your email...</CardTitle>
            </>
          )}
          {status === "success" && (
            <>
              <div className="mx-auto mb-4 text-4xl">✅</div>
              <CardTitle>Email verified!</CardTitle>
              <CardDescription>Redirecting you to sign in...</CardDescription>
            </>
          )}
          {status === "error" && (
            <>
              <div className="mx-auto mb-4 text-4xl">❌</div>
              <CardTitle>Verification failed</CardTitle>
              <CardDescription>
                This link is invalid or has expired. Please register again.
              </CardDescription>
            </>
          )}
        </CardHeader>
        {status === "error" && (
          <CardContent>
            <Button asChild className="w-full">
              <Link href="/register">Back to register</Link>
            </Button>
          </CardContent>
        )}
      </Card>
    </div>
  );
}
