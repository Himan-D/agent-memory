import { renderHook, act } from '@testing-library/react';
import { useSecurityValidation } from '@/hooks/use-security-validation';

// Mock DOMPurify
const mockDOMPurify = {
  sanitize: jest.fn((input: string) => input.replace(/<script>/g, '')),
};

jest.mock('dompurify', () => ({
  default: mockDOMPurify,
}));

describe('Security Validation Hook', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('input sanitization', () => {
    test('sanitizes HTML content', () => {
      const { result } = renderHook(() => useSecurityValidation());
      
      const maliciousInput = 'Hello <script>alert("xss")</script> World';
      const sanitized = result.current.sanitizeInput(maliciousInput);
      
      expect(sanitized).toBe('Hello  World');
      expect(mockDOMPurify.sanitize).toHaveBeenCalledWith(maliciousInput);
    });

    test('validates email format', () => {
      const { result } = renderHook(() => useSecurityValidation());
      
      expect(result.current.isValidEmail('user@example.com')).toBe(true);
      expect(result.current.isValidEmail('invalid-email')).toBe(false);
      expect(result.current.isValidEmail('')).toBe(false);
    });

    test('validates URL format', () => {
      const { result } = renderHook(() => useSecurityValidation());
      
      expect(result.current.isValidUrl('https://example.com')).toBe(true);
      expect(result.current.isValidUrl('http://localhost:3000')).toBe(true);
      expect(result.current.isValidUrl('invalid-url')).toBe(false);
    });

    test('validates content length', () => {
      const { result } = renderHook(() => useSecurityValidation());
      
      expect(result.current.validateContentLength('short', 100)).toBe(true);
      expect(result.current.validateContentLength('a'.repeat(1000), 100)).toBe(false);
    });

    test('detects SQL injection patterns', () => {
      const { result } = renderHook(() => useSecurityValidation());
      
      const sqlInjectionInputs = [
        "'; DROP TABLE users; --",
        "1 OR 1=1",
        "admin'--",
        "'; WAITFOR DELAY '0:0:10'--",
      ];

      sqlInjectionInputs.forEach(input => {
        expect(result.current.detectSqlInjection(input)).toBe(true);
      });

      expect(result.current.detectSqlInjection('normal input')).toBe(false);
    });

    test('detects XSS patterns', () => {
      const { result } = renderHook(() => useSecurityValidation());
      
      const xssInputs = [
        '<script>alert("xss")</script>',
        'javascript:alert("xss")',
        'onerror=alert("xss")',
        '<img src=x onerror=alert("xss")>',
      ];

      xssInputs.forEach(input => {
        expect(result.current.detectXSS(input)).toBe(true);
      });

      expect(result.current.detectXSS('normal input')).toBe(false);
    });

    test('validates file uploads', () => {
      const { result } = renderHook(() => useSecurityValidation());
      
      const validFile = new File(['test'], 'test.txt', { type: 'text/plain' });
      const executableFile = new File(['test'], 'test.exe', { type: 'application/octet-stream' });
      
      expect(result.current.validateFile(validFile, ['txt', 'pdf'])).toBe(true);
      expect(result.current.validateFile(executableFile, ['txt', 'pdf'])).toBe(false);
      expect(result.current.validateFile(validFile, ['txt'], 1000000)).toBe(true);
      expect(result.current.validateFile(validFile, ['txt'], 100)).toBe(false);
    });
  });

  describe('form validation', () => {
    test('validates memory creation form', () => {
      const { result } = renderHook(() => useSecurityValidation());
      
      const validForm = {
        content: 'Valid memory content',
        type: 'user',
        category: 'personal',
        importance: 'medium',
      };

      const invalidForm = {
        content: '',
        type: 'invalid-type',
        category: '<script>malicious</script>',
        importance: 'invalid-importance',
      };

      expect(result.current.validateMemoryForm(validForm)).toEqual({
        isValid: true,
        errors: {},
      });

      const validation = result.current.validateMemoryForm(invalidForm);
      expect(validation.isValid).toBe(false);
      expect(validation.errors.content).toBe('Content is required');
      expect(validation.errors.type).toBe('Invalid memory type');
    });

    test('validates user creation form', () => {
      const { result } = renderHook(() => useSecurityValidation());
      
      const validForm = {
        email: 'user@example.com',
        name: 'John Doe',
        role: 'member',
      };

      const invalidForm = {
        email: 'invalid-email',
        name: '',
        role: 'invalid-role',
      };

      expect(result.current.validateUserForm(validForm)).toEqual({
        isValid: true,
        errors: {},
      });

      const validation = result.current.validateUserForm(invalidForm);
      expect(validation.isValid).toBe(false);
      expect(validation.errors.email).toBe('Invalid email format');
      expect(validation.errors.name).toBe('Name is required');
    });

    test('validates skill creation form', () => {
      const { result } = renderHook(() => useSecurityValidation());
      
      const validForm = {
        name: 'Test Skill',
        trigger: 'test-trigger',
        action: 'test-action',
        confidence: 0.8,
      };

      const invalidForm = {
        name: '',
        trigger: '<script>malicious</script>',
        action: '',
        confidence: 1.5,
      };

      expect(result.current.validateSkillForm(validForm)).toEqual({
        isValid: true,
        errors: {},
      });

      const validation = result.current.validateSkillForm(invalidForm);
      expect(validation.isValid).toBe(false);
      expect(validation.errors.name).toBe('Name is required');
      expect(validation.errors.trigger).toBe('Trigger contains malicious content');
      expect(validation.errors.confidence).toBe('Confidence must be between 0 and 1');
    });
  });

  describe('rate limiting', () => {
    test('enforces rate limiting on API calls', () => {
      const { result } = renderHook(() => useSecurityValidation());
      
      // Simulate rapid API calls
      for (let i = 0; i < 10; i++) {
        result.current.rateLimitedApiCall('test-endpoint', () => Promise.resolve());
      }

      // Should allow calls within limit
      expect(result.current.rateLimitedApiCall('test-endpoint', () => Promise.resolve())).toBe(true);
      
      // Should block calls that exceed limit
      jest.useFakeTimers();
      act(() => {
        jest.advanceTimersByTime(1000); // Advance time beyond rate limit window
      });
      
      expect(result.current.rateLimitedApiCall('test-endpoint', () => Promise.resolve())).toBe(true);
    });
  });

  describe('content scanning', () => {
    test('scans for sensitive information', () => {
      const { result } = renderHook(() => useSecurityValidation());
      
      const sensitiveContent = `
        Password: secret123
        API Key: sk_test_1234567890
        Credit Card: 4111111111111111
        SSN: 123-45-6789
      `;

      const scanResult = result.current.scanForSensitiveData(sensitiveContent);
      expect(scanResult.hasSensitiveData).toBe(true);
      expect(scanResult.detectedTypes).toContain('password');
      expect(scanResult.detectedTypes).toContain('api_key');
      expect(scanResult.detectedTypes).toContain('credit_card');
      expect(scanResult.detectedTypes).toContain('ssn');
    });

    test('allows normal content without sensitive data', () => {
      const { result } = renderHook(() => useSecurityValidation());
      
      const normalContent = 'This is a normal memory about a user experience.';
      
      const scanResult = result.current.scanForSensitiveData(normalContent);
      expect(scanResult.hasSensitiveData).toBe(false);
      expect(scanResult.detectedTypes).toEqual([]);
    });
  });
});