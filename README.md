# GoreeShield

**GoreeShield** is GoreeCloud's proposed privacy platform for its self-hosted ecosystem. Its mission is to help people understand, govern, and verify how their personal information is collected, used, shared, retained, and deleted.

> **Status:** Planning and repository foundation only. No feature, API, native app, security property, legal compliance, or deployment is yet verified.

## Proposed feature families

| Family | Purpose |
| --- | --- |
| **GoreeShield Center** | Central privacy experience and status |
| **GoreeShield Choices** | Granular opt-in, preferences, withdrawal, and consent receipts |
| **GoreeShield Lens** | Processing inventory, purposes, recipients, and data-flow visibility |
| **GoreeShield Boundary** | Optional integrations, permissions, and sharing controls |
| **GoreeShield Rights** | Data access, correction, portability, and erasure requests |
| **GoreeShield Lifecycle** | Minimization, retention, holds, and verified deletion |
| **GoreeShield Veil** | Optional tracking protection where native platform capabilities permit |
| **GoreeShield Family** | Privacy-respecting household controls, without covert monitoring |

Family names are proposals, not confirmed independent products.

## Boundaries

- GoreeShield owns privacy choices and data-governance coordination.
- **Wardveil** owns malware, threats, vulnerabilities, security monitoring, and incident response.
- **GoreeCloud Vault** owns passwords, credentials, keys, and secrets.
- **GoreeCloud Identity** owns identity, authentication, and SSO.
- **GoreeCloud UI (GC UI)** is GoreeCloud's approved design authority.

GoreeShield should integrate through least-privilege, versioned interfaces, not duplicate these systems.

## Design commitments

Privacy-invasive optional features start **off** until the user knowingly opts in. Choices must be granular, reversible, and reflected in actual processing. Data stays with its owning product where possible. Do not collect raw personal payloads centrally by default. Verify actual outcomes before reporting access, correction, or deletion complete.

The planned server and apps must be genuinely native to each declared target operating system; browser wrappers are not native deliverables. Claims of support require target-specific evidence.

## Planning documents

- [Vision and scope](docs/vision.md)
- [Feature families](docs/feature-families.md)
- [Technical architecture](docs/architecture.md)
- [Privacy requirements](docs/privacy-model.md)
- [Roadmap](docs/roadmap.md)

This **public** repository must not contain credentials, private keys, production data, private infrastructure details, or private incident information. Project planning documents are not a substitute for GoreeCloud's authoritative governance. Licensing and first supported platforms are to be decided before implementation.
