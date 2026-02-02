## 2024-07-25 - Stored plain text passwords

**Vulnerability:** Passwords were being stored and checked in plaintext directly from the config file.
**Learning:** The application's `authenticationMiddleware` was performing a direct string comparison (`password == item.Password`), which is highly insecure. When implementing a breaking security change like this, it's crucial to provide a seamless migration path for existing users. My initial fix broke authentication for users with plaintext passwords. The corrected approach hashes plaintext passwords on the fly, logs a warning, and allows the application to continue functioning, which is a much better user experience.
**Prevention:** Always hash passwords using a strong, well-vetted algorithm like bcrypt before storing or comparing them. Never store plaintext credentials. When implementing breaking security changes, consider the user experience and provide a migration path or graceful handling of old configurations.
