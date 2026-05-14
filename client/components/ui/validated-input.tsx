import * as React from "react";
import { AlertCircle, CheckCircle } from "lucide-react";
import { cn } from "@/lib/utils";

export interface ValidatedInputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  error?: string | null;
  touched?: boolean;
  label?: string;
  showValidIcon?: boolean;
}

const ValidatedInput = React.forwardRef<HTMLInputElement, ValidatedInputProps>(
  ({ className, type, error, touched, label, showValidIcon = true, ...props }, ref) => {
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
            type={type}
            className={cn(
              "flex h-10 w-full rounded-md border bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
              hasError
                ? "border-destructive focus-visible:ring-destructive pr-10"
                : isValid && showValidIcon
                ? "border-green-500 focus-visible:ring-green-500 pr-10"
                : "border-input pr-3",
              className
            )}
            ref={ref}
            {...props}
          />
          {hasError && (
            <AlertCircle className="absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-destructive" />
          )}
          {isValid && showValidIcon && (
            <CheckCircle className="absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-green-500" />
          )}
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
ValidatedInput.displayName = "ValidatedInput";

export { ValidatedInput };