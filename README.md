# GoreeShield

**GoreeCloud's privacy controls, privacy protection, and data-governance platform.**

> **Status:** Founder-designated privacy platform and one of the five Mandatory Integrated Platform Systems (MIPS). This README defines the intended product responsibilities, not implemented features. Working privacy controls, integrations, and production deployment have not yet been independently verified.

## Purpose

GoreeShield is intended to help people understand, limit, and control how their personal information is collected, processed, stored, retained, accessed, and shared across GoreeCloud applications and authorized integrations.

Its focus is **privacy**, not cybersecurity. Privacy-by-default practices and meaningful user choice should guide its design. A service must continue to meet applicable privacy requirements even if GoreeShield is unavailable or has not yet been integrated.

## Intended capabilities

The following are **areas for product design and validation**, not claims of completed functionality or final API contracts:

- **Privacy preferences and controls:** Clear, granular choices about optional data collection, processing, visibility, and sharing.
- **Consent and transparency:** Understandable privacy notices, consent records where appropriate, withdrawal controls, and explainable privacy decisions.
- **Data minimization and purpose limitation:** Collect and use only data needed for a specified, authorized purpose.
- **Tracking and third-party data controls:** Give users appropriate visibility and choices concerning analytics, trackers, third-party disclosures, and connected services.
- **Retention and lifecycle controls:** Support defined retention periods and governed deletion, export, and portability workflows through data-owning services.
- **Sharing and disclosure management:** Control which recipients, services, or integrations can receive particular categories of personal information.
- **Privacy policy coordination:** Provide well-defined privacy rules and decision interfaces for participating products, while those products remain responsible for enforcing their own data handling.
- **Privacy accountability:** Provide proportionate, privacy-preserving records of consent, processing purposes, data flows, and policy decisions without excessive logging of sensitive content.

These capabilities require individual requirements, ownership decisions, accessible user experiences, tests, and evidence before being described as implemented.

## Responsibility boundaries

| Platform | Distinct responsibility |
| --- | --- |
| **GoreeShield** | Privacy controls, privacy preferences, consent, data minimization, retention, disclosure, and personal-data governance |
| **Wardveil** | Cybersecurity, threat detection, malware and vulnerability scanning, network/application defense, security monitoring, and incident response |
| **GoreeCloud Identity** | Identity, authentication, and identity-related integration |
| **GoreeCloud Mesh** | Authorized interoperability, API/MCP connections, data exchange, and orchestration; it must respect applicable privacy controls |
| **GoreeCloud UI** | Shared design system and accessible interaction patterns |

GoreeShield **does not own general cybersecurity or protective threat-defense functions**. Those belong to Wardveil. Security safeguards remain important to privacy, but that overlap does not make GoreeShield a malware scanner, firewall, intrusion detection system, or incident-response service.

GoreeShield also does not replace each application's duty to protect data, implement its own authorized operations, or honor user preferences. It may coordinate privacy decisions through approved interfaces when those are specified and verified.

## Development checkpoint — November 6, 2026

GoreeShield's **experimental, Linux-local privacy Choices implementation is being developed in draft pull requests**, not on this repository's default branch. [Draft PR #6](https://github.com/GoreeCloud/goreeshield/pull/6) begins a synthetic purpose and consent proof of concept; [draft PR #30](https://github.com/GoreeCloud/goreeshield/pull/30) adds a fail-closed regression for unsupported state schemas on top of preceding draft work. PR #30 targets another feature branch, not `main`. Its head's **Linux prototype checks** reported success when inspected on November 6, 2026, but that check is not end-to-end or release acceptance.

The experimental work must remain subject to dependency-ordered review, code and privacy-security testing, source-owned enforcement checks, genuine authenticated subject binding, independent outcome verification, and target-OS native acceptance before any integration or release claim. Track outstanding work in [privacy architecture issue #3](https://github.com/GoreeCloud/goreeshield/issues/3), [native MVP issue #4](https://github.com/GoreeCloud/goreeshield/issues/4), and [Choices issue #5](https://github.com/GoreeCloud/goreeshield/issues/5). No real personal-information processing, operational source adapters, production identity integration, or deployed service is verified by these draft tests.

## Governance and integration

GoreeShield is one of GoreeCloud's [five Mandatory Integrated Platform Systems (MIPS)](https://github.com/GoreeCloud/MIPS). Applicable GoreeCloud products must assess and verify their GoreeShield integration, or document a defensible non-applicability rationale, before satisfying the MIPS prerequisite for **Anchor** release eligibility.

The governing requirements and documentation authority reside in the authorized GoreeCloud Google Drive records. This repository is the GoreeShield product source; proposals here are not proof of deployed capabilities or automatically approved platform contracts.

## Verification standard

Before claiming a privacy capability is functional, verify its implementation and representative behavior, including correct allowed/denied data flows, consent and preference changes, service-side enforcement, failure handling, accessibility, and appropriate privacy-preserving audit evidence.

Do not treat a README, mock UI, placeholder API, or future integration plan as an operational privacy feature.

## License

The GoreeCloud-controlled source and documentation currently contained in this repository are provided under the [Apache License, Version 2.0](LICENSE). The existing `main` branch currently contains product and engineering documentation, not an operational privacy service; experimental Choices code in separate unmerged pull requests is subject to independent dependency, provenance, contributor-rights and acceptance review before incorporation. Any future third-party code, licensed content, data, icons, fonts or dependencies retain their separate applicable licenses and notices.
