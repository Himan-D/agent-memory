## Security Pattern: Command Injection

*   **Vulnerability:** Directly passing user-supplied input to `sh -c` or similar shell executors creates a high-severity command injection vulnerability. An attacker can use shell metacharacters (e.g., `;`, `&`, `|`, `$()`, `` ` ``) to execute arbitrary commands.
*   **Fix:** Avoid invoking a shell. Parse the command line string into a command and arguments list and execute it directly via `exec.Command(cmd, args...)`. For parsing shell-like strings safely without evaluating them, use libraries like `github.com/mattn/go-shellwords`.
*   **Code Example (Go):**
    *   **Vulnerable:** `exec.Command("sh", "-c", userInput)`
    *   **Secure:**
        ```go
        parts, err := shellwords.Parse(userInput)
        // ... error handling ...
        cmd := exec.Command(parts[0], parts[1:]...)
        ```
