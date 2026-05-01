# Security policy

## Supported versions

The latest `1.x` release is supported for security fixes. Older releases may receive no further updates.

## Reporting a vulnerability

If you find a security issue in `ma`, please open a private security advisory or contact the maintainer directly instead of filing a public issue with exploit details.

When you report an issue, include:

1. the affected version or commit
2. the impact and any likely attack preconditions
3. a minimal reproduction, ideally using synthetic data instead of real secrets

`ma` is designed to stay local-only and to reject sensitive paths such as `.env`, `.ssh`, and symlinks that resolve to protected targets. Reports that show a path around those guarantees are especially valuable.
