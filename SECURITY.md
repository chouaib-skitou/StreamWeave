# Security Policy

## Scope

This repository is a portfolio and learning project. It uses simulated identity and payment data only. Never commit passwords, tokens, private keys, credentials, real cardholder data, or production configuration.

## Reporting a vulnerability

Do not open a public issue for a suspected vulnerability or include secrets and exploit details in a pull request. Report it privately to the repository owner through GitHub's private vulnerability reporting channel when enabled, or through the private contact method configured for this repository.

## Security expectations

- Validate authentication, authorization, resource ownership, and service identity at the appropriate boundary.
- Treat Kafka delivery as at-least-once and make consumers idempotent.
- Do not use Redis as durable security or business state.
- Review dependency, container, infrastructure, and contract changes before merging.
