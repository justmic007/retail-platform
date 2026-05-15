"use client";

import { useState, useCallback } from "react";

export interface ValidationRule {
  required?: boolean;
  minLength?: number;
  maxLength?: number;
  pattern?: RegExp;
  custom?: (value: string) => string | null;
}

export interface FieldState {
  value: string;
  error: string | null;
  touched: boolean;
}

export interface FormState {
  [key: string]: FieldState;
}

export function useFormValidation(initialValues: Record<string, string>, rules: Record<string, ValidationRule>) {
  const [formState, setFormState] = useState<FormState>(() => {
    const state: FormState = {};
    Object.keys(initialValues).forEach(key => {
      state[key] = {
        value: initialValues[key],
        error: null,
        touched: false
      };
    });
    return state;
  });

  const validateField = useCallback((name: string, value: string): string | null => {
    const rule = rules[name];
    if (!rule) return null;

    if (rule.required && !value.trim()) {
      return "This field is required";
    }

    if (rule.minLength && value.length < rule.minLength) {
      return `Must be at least ${rule.minLength} characters`;
    }

    if (rule.maxLength && value.length > rule.maxLength) {
      return `Must be no more than ${rule.maxLength} characters`;
    }

    if (rule.pattern && !rule.pattern.test(value)) {
      if (name === "email") return "Please enter a valid email address";
      return "Invalid format";
    }

    if (rule.custom) {
      return rule.custom(value);
    }

    return null;
  }, [rules]);

  const setValue = useCallback((name: string, value: string) => {
    setFormState(prev => ({
      ...prev,
      [name]: {
        ...prev[name],
        value,
        error: prev[name].touched ? validateField(name, value) : null
      }
    }));
  }, [validateField]);

  const setTouched = useCallback((name: string) => {
    setFormState(prev => ({
      ...prev,
      [name]: {
        ...prev[name],
        touched: true,
        error: validateField(name, prev[name].value)
      }
    }));
  }, [validateField]);

  const validateAll = useCallback(() => {
    const newState = { ...formState };
    let isValid = true;

    Object.keys(newState).forEach(name => {
      const error = validateField(name, newState[name].value);
      newState[name] = {
        ...newState[name],
        touched: true,
        error
      };
      if (error) isValid = false;
    });

    setFormState(newState);
    return isValid;
  }, [formState, validateField]);

  const reset = useCallback(() => {
    const state: FormState = {};
    Object.keys(initialValues).forEach(key => {
      state[key] = {
        value: initialValues[key],
        error: null,
        touched: false
      };
    });
    setFormState(state);
  }, [initialValues]);

  const getValues = useCallback(() => {
    const values: Record<string, string> = {};
    Object.keys(formState).forEach(key => {
      values[key] = formState[key].value;
    });
    return values;
  }, [formState]);

  return {
    formState,
    setValue,
    setTouched,
    validateAll,
    reset,
    getValues,
    isValid: Object.values(formState).every(field => !field.error)
  };
}