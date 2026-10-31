## 2026-07-03 - Unchecked crypto/rand Returns
**Vulnerability:** Ignored errors from `crypto/rand.Read` when generating IDs, tokens, and secrets.
**Learning:** If the random generator fails, it leaves the buffer with zero bytes or uninitialized memory, leading to predictable secrets that can be easily guessed, resulting in potential auth bypass or session hijacking.
**Prevention:** Always explicitly check the error returned by `crypto/rand.Read` and fail securely (e.g., panic or return error) to ensure cryptographic safety.
