## 2024-07-25 - Stored plain text passwords
**Vulnerability:** Passwords were being stored and checked in plaintext directly from the config file.
**Learning:** The application's `authenticationMiddleware` was performing a direct string comparison (`password == item.Password`), which is highly insecure. When implementing a breaking security change like this, it's crucial to provide a seamless migration path for existing users. My initial fix broke authentication for users with plaintext passwords. The corrected approach hashes plaintext passwords on the fly, logs a warning, and allows the application to continue functioning, which is a much better user experience.
**Prevention:** Always hash passwords using a strong, well-vetted algorithm like bcrypt before storing or comparing them. Never store plaintext credentials. When implementing breaking security changes, consider the user experience and provide a migration path or graceful handling of old configurations.

## 2026-01-25 - Missing CSRF Protection
**Vulnerability:** The application was vulnerable to CSRF attacks because it relied solely on Basic Auth (which browsers send automatically) for POST requests without any CSRF token verification.
**Learning:** Even stateless applications using Basic Auth are vulnerable to CSRF if they don't verify the origin of the request for state-changing operations. Double Submit Cookie is a viable pattern for such applications.
**Prevention:** Implemented a CSRF middleware using the Double Submit Cookie pattern. It sets a random token in a cookie and requires the same token to be present in the `X-CSRF-Token` header for state-changing requests. Updated the frontend to inject this header into HTMX requests.
