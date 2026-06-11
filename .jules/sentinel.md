## 2024-06-11 - [CRITICAL] GitHub Webhook Signature Verification Bypass
**Vulnerability:** Found `verifyGitHubSignature` in `internal/connectors/github.go` was returning `true` unconditionally, completely bypassing webhook signature validation.
**Learning:** Stubbed security functions (likely left over from initial development or testing) can easily be forgotten and deployed to production, resulting in critical vulnerabilities allowing unauthorized actors to forge requests.
**Prevention:** Implement security functionality at the same time as the features they protect. Write automated tests that explicitly verify that unauthorized or forged requests fail.
