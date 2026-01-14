## 2024-07-25 - Stored plain text passwords
**Vulnerability:** Passwords were being stored and checked in plaintext directly from the config file.
**Learning:** The application's `authenticationMiddleware` was performing a direct string comparison (`password == item.Password`), which is highly insecure.
**Prevention:** Always hash passwords using a strong, well-vetted algorithm like bcrypt before storing or comparing them. Never store plaintext credentials.
