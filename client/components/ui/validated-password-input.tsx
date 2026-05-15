"use client";

import * as React from "react";
import { Eye, EyeOff, AlertCircle, CheckCircle } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

export interface ValidatedPasswordInputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  error?: string | null;
  touched?: boolean;
  label?: string;
  showValidIcon?: boolean;
}

const ValidatedPasswordInput = React.forwardRef<HTMLInputElement, ValidatedPasswordInputProps>(
  ({ className, error, touched, label, showValidIcon = true, ...props }, ref) => {
    const [showPassword, setShowPassword] = React.useState(false);
    const hasError = touched && error;
    const isValid = touched && !error && props.value;

    return (
      <div className="space-y-2">
        {label && (
          <label htmlFor={props.id} className="text-sm font-medium">
            {label}
          </label>
        )}
        <div className="relative">
          <input
            type={showPassword ? "text" : "password"}
            className={cn(
              "flex h-10 w-full rounded-md border bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 pr-20",
              hasError
                ? "border-destructive focus-visible:ring-destructive"
                : isValid && showValidIcon
                ? "border-green-500 focus-visible:ring-green-500"
                : "border-input",
              className
            )}
            ref={ref}
            {...props}
          />
          <div className="absolute right-0 top-0 h-full flex items-center">
            {hasError && (
              <AlertCircle className="h-4 w-4 text-destructive mr-2" />
            )}
            {isValid && showValidIcon && (
              <CheckCircle className="h-4 w-4 text-green-500 mr-2" />
            )}
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-full px-3 py-2 hover:bg-transparent"
              onClick={() => setShowPassword(!showPassword)}
              tabIndex={-1}
            >
              {showPassword ? (
                <EyeOff className="h-4 w-4 text-muted-foreground" />
              ) : (
                <Eye className="h-4 w-4 text-muted-foreground" />
              )}
            </Button>
          </div>
        </div>
        {hasError && (
          <p className="text-sm text-destructive flex items-center gap-1">
            <AlertCircle className="h-3 w-3" />
            {error}
          </p>
        )}
      </div>
    );
  }
);
ValidatedPasswordInput.displayName = "ValidatedPasswordInput";

export { ValidatedPasswordInput };