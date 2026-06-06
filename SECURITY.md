# Security Policy

## Supported Versions

This is a learning project, so security fixes should target the current `main` branch.

## Reporting A Vulnerability

Do not open a public issue for sensitive reports. Contact the maintainer privately, or use the private reporting mechanism of the hosting platform if one is available.

Include:

- A short description of the issue.
- Reproduction steps.
- Impact and affected routes or components.
- Suggested fix, if known.

## Security Posture

- The production image is distroless and runs without a shell.
- The server runs as a non-root user in the production container.
- `DATABASE_URL` is required and should come from the deployment environment or secret manager.
- `.env` is ignored and must not be committed.
- CORS is configurable through `CORS_ALLOWED_ORIGINS`; production deployments should avoid `*` unless the app is intentionally public and same-origin controls are handled elsewhere.

Because this is my first Go project, the security setup is intentionally simple and educational. Please share practical improvements if you notice anything that could be safer.
