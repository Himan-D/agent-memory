"use client";

import { useState, useCallback } from "react";

interface ValidationResult {
  isValid: boolean;
  errors: Record<string, string>;
}

interface SecurityScanResult {
  hasSensitiveData: boolean;
  detectedTypes: string[];
  warnings: string[];
}

export function useSecurityValidation() {
  const [rateLimitMap, setRateLimitMap] = useState<Record<string, number[]>>({});

  const sanitizeInput = useCallback((input: string): string => {
    if (typeof input !== "string") return "";
    // Remove null bytes
    let sanitized = input.replace(/\0/g, "");
    // Remove control characters except newlines and tabs
    sanitized = sanitized.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, "");
    return sanitized;
  }, []);

  const isValidEmail = useCallback((email: string): boolean => {
    if (!email || typeof email !== "string") return false;
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return email.length <= 254 && emailRegex.test(email);
  }, []);

  const isValidUrl = useCallback((url: string): boolean => {
    if (!url || typeof url !== "string") return false;
    try {
      const parsed = new URL(url);
      return ["http:", "https:"].includes(parsed.protocol);
    } catch {
      return false;
    }
  }, []);

  const validateContentLength = useCallback((content: string, maxLength: number): boolean => {
    if (!content || typeof content !== "string") return false;
    return content.length <= maxLength;
  }, []);

  const detectSqlInjection = useCallback((input: string): boolean => {
    if (!input || typeof input !== "string") return false;
    const sqlPatterns = [
      /(\b(SELECT|INSERT|UPDATE|DELETE|DROP|ALTER|CREATE|TRUNCATE)\b)/i,
      /('|")?\s*(OR|AND)\s+\d+\s*=\s*\d+/i,
      /--\s*$/,
      /;\s*(DROP|DELETE|INSERT|UPDATE|SELECT)/i,
      /\b(WAITFOR|EXEC|EXECUTE|xp_|sp_)\b/i,
      /'\s*;\s*--/,
      /\bUNION\s+(ALL\s+)?SELECT\b/i,
    ];
    return sqlPatterns.some((pattern) => pattern.test(input));
  }, []);

  const detectXSS = useCallback((input: string): boolean => {
    if (!input || typeof input !== "string") return false;
    const xssPatterns = [
      /<script[\s>]/i,
      /javascript\s*:/i,
      /on\w+\s*=\s*["']?[^"'>]*(["']?\s*>|&)/i,
      /<\s*img[^>]+onerror\s*=/i,
      /<\s*svg[^>]+onload\s*=/i,
      /<\s*iframe/i,
      /<\s*object/i,
      /<\s*embed/i,
      /<\s*link[^>]+href\s*=\s*["']?javascript/i,
      /<\s*form[^>]+action\s*=\s*["']?javascript/i,
      /<\s*a[^>]+href\s*=\s*["']?javascript/i,
      /<\s*input[^>]+formaction\s*=\s*["']?javascript/i,
      /<\s*button[^>]+formaction\s*=\s*["']?javascript/i,
      /<\s*textarea[^>]+formaction\s*=\s*["']?javascript/i,
      /<\s*select[^>]+formaction\s*=\s*["']?javascript/i,
      /data\s*:\s*text\/html/i,
      /vbscript\s*:/i,
      /<\s*math[^>]+script/i,
    ];
    return xssPatterns.some((pattern) => pattern.test(input));
  }, []);

  const validateFile = useCallback(
    (file: File, allowedExtensions: string[], maxSizeBytes?: number): boolean => {
      if (!file || !(file instanceof File)) return false;

      const ext = file.name.split(".").pop()?.toLowerCase();
      if (ext && !allowedExtensions.includes(ext)) return false;

      const dangerousExtensions = ["exe", "bat", "cmd", "sh", "ps1", "msi", "dll", "so", "bin", "scr", "com"];
      if (ext && dangerousExtensions.includes(ext)) return false;

      if (maxSizeBytes && file.size > maxSizeBytes) return false;

      return true;
    },
    []
  );

  const validateMemoryForm = useCallback((data: Record<string, unknown>): ValidationResult => {
    const errors: Record<string, string> = {};

    if (!data.content || typeof data.content !== "string" || !data.content.trim()) {
      errors.content = "Content is required";
    } else if (data.content.length > 50000) {
      errors.content = "Content must be less than 50,000 characters";
    }

    if (!data.type || typeof data.type !== "string") {
      errors.type = "Type is required";
    } else if (!["conversation", "session", "user", "org"].includes(String(data.type))) {
      errors.type = "Invalid memory type";
    }

    if (data.category && typeof data.category === "string" && data.category.length > 100) {
      errors.category = "Category must be less than 100 characters";
    }

    if (data.tags && Array.isArray(data.tags) && data.tags.length > 20) {
      errors.tags = "Maximum 20 tags allowed";
    }

    if (data.importance && !["critical", "high", "medium", "low"].includes(String(data.importance))) {
      errors.importance = "Invalid importance level";
    }

    // Check for sensitive data
    if (data.content && typeof data.content === "string") {
      if (detectSqlInjection(data.content)) {
        errors.content = "Content contains potentially dangerous patterns";
      }
    }

    return {
      isValid: Object.keys(errors).length === 0,
      errors,
    };
  }, [detectSqlInjection]);

  const validateUserForm = useCallback((data: Record<string, unknown>): ValidationResult => {
    const errors: Record<string, string> = {};

    if (!data.email || typeof data.email !== "string" || !isValidEmail(data.email)) {
      errors.email = "Invalid email format";
    }

    if (!data.name || typeof data.name !== "string" || !data.name.trim()) {
      errors.name = "Name is required";
    } else if (data.name.length > 200) {
      errors.name = "Name must be less than 200 characters";
    }

    if (data.role && !["admin", "member", "viewer"].includes(String(data.role))) {
      errors.role = "Invalid role";
    }

    if (data.status && !["active", "inactive"].includes(String(data.status))) {
      errors.status = "Invalid status";
    }

    return {
      isValid: Object.keys(errors).length === 0,
      errors,
    };
  }, [isValidEmail]);

  const validateSkillForm = useCallback((data: Record<string, unknown>): ValidationResult => {
    const errors: Record<string, string> = {};

    if (!data.name || typeof data.name !== "string" || !data.name.trim()) {
      errors.name = "Name is required";
    } else if (data.name.length > 200) {
      errors.name = "Name must be less than 200 characters";
    }

    if (data.trigger && typeof data.trigger === "string" && detectXSS(data.trigger)) {
      errors.trigger = "Trigger contains malicious content";
    }

    if (data.action && typeof data.action === "string" && detectXSS(data.action)) {
      errors.action = "Action contains malicious content";
    }

    if (data.confidence !== undefined) {
      const conf = Number(data.confidence);
      if (isNaN(conf) || conf < 0 || conf > 1) {
        errors.confidence = "Confidence must be between 0 and 1";
      }
    }

    if (data.usage_count !== undefined) {
      const count = Number(data.usage_count);
      if (isNaN(count) || count < 0) {
        errors.usage_count = "Usage count must be a non-negative number";
      }
    }

    return {
      isValid: Object.keys(errors).length === 0,
      errors,
    };
  }, [detectXSS]);

  const validateEntityForm = useCallback((data: Record<string, unknown>): ValidationResult => {
    const errors: Record<string, string> = {};

    if (!data.name || typeof data.name !== "string" || !data.name.trim()) {
      errors.name = "Name is required";
    } else if (data.name.length > 500) {
      errors.name = "Name must be less than 500 characters";
    }

    if (!data.type || typeof data.type !== "string" || !data.type.trim()) {
      errors.type = "Type is required";
    }

    return {
      isValid: Object.keys(errors).length === 0,
      errors,
    };
  }, []);

  const validateWebhookForm = useCallback((data: Record<string, unknown>): ValidationResult => {
    const errors: Record<string, string> = {};

    if (!data.url || typeof data.url !== "string") {
      errors.url = "URL is required";
    } else if (!isValidUrl(data.url)) {
      errors.url = "Invalid URL format";
    }

    if (data.events && Array.isArray(data.events) && data.events.length === 0) {
      errors.events = "At least one event is required";
    }

    return {
      isValid: Object.keys(errors).length === 0,
      errors,
    };
  }, [isValidUrl]);

  const validateChainForm = useCallback((data: Record<string, unknown>): ValidationResult => {
    const errors: Record<string, string> = {};

    if (!data.name || typeof data.name !== "string" || !data.name.trim()) {
      errors.name = "Name is required";
    }

    if (!data.trigger || typeof data.trigger !== "string" || !data.trigger.trim()) {
      errors.trigger = "Trigger is required";
    }

    if (data.steps && Array.isArray(data.steps) && data.steps.length === 0) {
      errors.steps = "At least one step is required";
    }

    return {
      isValid: Object.keys(errors).length === 0,
      errors,
    };
  }, []);

  const validateAgentForm = useCallback((data: Record<string, unknown>): ValidationResult => {
    const errors: Record<string, string> = {};

    if (!data.name || typeof data.name !== "string" || !data.name.trim()) {
      errors.name = "Name is required";
    } else if (data.name.length > 200) {
      errors.name = "Name must be less than 200 characters";
    }

    if (data.status && !["active", "inactive", "suspended"].includes(String(data.status))) {
      errors.status = "Invalid status";
    }

    return {
      isValid: Object.keys(errors).length === 0,
      errors,
    };
  }, []);

  const validateProjectForm = useCallback((data: Record<string, unknown>): ValidationResult => {
    const errors: Record<string, string> = {};

    if (!data.name || typeof data.name !== "string" || !data.name.trim()) {
      errors.name = "Name is required";
    } else if (data.name.length > 500) {
      errors.name = "Name must be less than 500 characters";
    }

    return {
      isValid: Object.keys(errors).length === 0,
      errors,
    };
  }, []);

  const rateLimitedApiCall = useCallback(
    (endpoint: string, fn: () => Promise<any>, maxCalls: number = 10, windowMs: number = 1000): boolean => {
      const now = Date.now();
      setRateLimitMap((prev) => {
        const calls = (prev[endpoint] || []).filter((t) => now - t < windowMs);
        if (calls.length >= maxCalls) return prev;
        return { ...prev, [endpoint]: [...calls, now] };
      });
      return true;
    },
    []
  );

  const scanForSensitiveData = useCallback((content: string): SecurityScanResult => {
    if (!content || typeof content !== "string") {
      return { hasSensitiveData: false, detectedTypes: [], warnings: [] };
    }

    const detections: { type: string; pattern: RegExp }[] = [
      { type: "password", pattern: /(?:password|passwd|pwd)\s*[:=]\s*["']?([^"'\s]+)/i },
      { type: "api_key", pattern: /(?:api[_-]?key|apikey)\s*[:=]\s*["']?([a-zA-Z0-9_-]{20,})/i },
      { type: "secret_key", pattern: /(?:secret[_-]?key)\s*[:=]\s*["']?([a-zA-Z0-9_-]{20,})/i },
      { type: "credit_card", pattern: /\b(?:\d{4}[\s-]?){3}\d{4}\b/ },
      { type: "ssn", pattern: /\b\d{3}-\d{2}-\d{4}\b/ },
      { type: "email", pattern: /[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}/ },
      { type: "phone", pattern: /\b\d{3}[-.]?\d{3}[-.]?\d{4}\b/ },
      { type: "token", pattern: /(?:access[_-]?token|auth[_-]?token)\s*[:=]\s*["']?([a-zA-Z0-9._-]+)/i },
      { type: "private_key", pattern: /-----BEGIN\s+(RSA |EC |DSA )?PRIVATE KEY-----/ },
      { type: "ip_address", pattern: /\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b/ },
    ];

    const detectedTypes = new Set<string>();
    const warnings: string[] = [];

    detections.forEach(({ type, pattern }) => {
      if (pattern.test(content)) {
        detectedTypes.add(type);
        warnings.push(`Potential ${type.replace(/_/g, " ")} detected in content`);
      }
    });

    return {
      hasSensitiveData: detectedTypes.size > 0,
      detectedTypes: Array.from(detectedTypes),
      warnings,
    };
  }, []);

  return {
    sanitizeInput,
    isValidEmail,
    isValidUrl,
    validateContentLength,
    detectSqlInjection,
    detectXSS,
    validateFile,
    validateMemoryForm,
    validateUserForm,
    validateSkillForm,
    validateEntityForm,
    validateWebhookForm,
    validateChainForm,
    validateAgentForm,
    validateProjectForm,
    rateLimitedApiCall,
    scanForSensitiveData,
  };
}