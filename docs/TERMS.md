# PayKit — Terms of Use

**Effective Date:** July 2026  
**Author:** Moses Wuruem Justin  
**Contact:** Available via GitHub Issues or the repository contact page

---

## 1. Acceptance

By downloading, installing, deploying, or using PayKit in any environment — development, staging, or production — you agree to these Terms of Use. If you do not agree, do not use the software.

---

## 2. What PayKit Is

PayKit is an open-source, self-hosted webhook ingestion engine distributed under the MIT License. It is software you run on your own infrastructure. The author does not operate PayKit on your behalf, does not have access to your data, and does not control your deployment.

---

## 3. Self-Hosted Responsibility

Because PayKit runs entirely on infrastructure you control:

- **You are responsible for your server.** Uptime, performance, security patches, and backups are your responsibility, not the author's.
- **You are responsible for your data.** All transactions, merchant records, delivery logs, and payer data stored by PayKit live in your PostgreSQL database. The author has no access to this data.
- **You are responsible for your secrets.** Your `MOMO_WEBHOOK_SECRET`, API keys, and database credentials must be protected by you. Do not commit them to version control.
- **You are responsible for compliance.** Ensure your use of PayKit complies with MTN MoMo's developer terms, your local financial data regulations (including Rwanda's Law No. 058/2021 on data protection and Uganda's Data Protection and Privacy Act), and any other applicable laws in your jurisdiction.

---

## 4. Subscription Tiers and Enforcement

PayKit ships with built-in subscription tier logic (Free, Starter, Growth, Enterprise). These tiers are enforced locally within your own deployment:

- Tier limits (`max_monthly_calls`) are stored in your database and enforced by the engine middleware.
- The author does not remotely monitor, audit, or enforce these limits.
- If you offer PayKit as a service to third parties, you are responsible for setting, communicating, and enforcing fair use limits with those parties.
- Monthly call counters reset on the 1st of each calendar month via the built-in reset job.

---

## 5. MTN MoMo Integration

PayKit is an independent tool and is not affiliated with, endorsed by, or officially partnered with MTN Group or any of its subsidiaries. The MTN MoMo API, webhook format, and authentication mechanisms are subject to change by MTN without notice. The author makes no guarantee that PayKit will remain compatible with future changes to the MTN MoMo API. It is your responsibility to test compatibility and update your deployment accordingly.

---

## 6. No Warranty

PayKit is provided "as is" without warranty of any kind, express or implied. This includes but is not limited to:

- No guarantee of uninterrupted or error-free operation
- No guarantee that webhook deliveries will succeed in all network conditions
- No guarantee of compatibility with all versions of MTN MoMo's API
- No guarantee that the Dead Letter Queue will recover all failed deliveries in all failure scenarios

Use PayKit in production only after thorough testing in a sandbox environment.

---

## 7. Limitation of Liability

To the maximum extent permitted by applicable law, the author shall not be liable for:

- Lost revenue, missed payments, or unfulfilled orders resulting from webhook delivery failures
- Data loss resulting from database misconfiguration, hardware failure, or improper backup practices
- Security incidents resulting from misconfigured secrets, exposed endpoints, or failure to apply security updates
- Any indirect, incidental, special, or consequential damages arising from use of the software

Your use of PayKit in a live payment environment is entirely at your own risk.

---

## 8. Permitted Use

Under the MIT License, you are free to:

- Use PayKit for personal, commercial, or enterprise purposes
- Modify the source code to fit your needs
- Distribute copies of PayKit
- Build products and services on top of PayKit
- Offer PayKit to your own customers as a hosted or self-hosted solution

---

## 9. Prohibited Use

You may not use PayKit to:

- Process payments in violation of MTN MoMo's developer terms of service
- Facilitate fraudulent transactions, money laundering, or any illegal financial activity
- Bypass, circumvent, or disable the built-in security mechanisms (HMAC verification, replay protection, timestamp validation) in production environments
- Misrepresent PayKit as an officially licensed MTN product or integration

---

## 10. Privacy

PayKit is designed with privacy in mind:

- Payer MSISDNs (phone numbers) are SHA-256 hashed before storage — plaintext phone numbers are never persisted.
- Raw webhook payloads are stored in `transactions.raw_payload` for audit purposes. You are responsible for managing access to this data and complying with applicable data retention laws.
- If you collect any personal data through your use of PayKit, you are solely responsible for obtaining the necessary consents and implementing required data protection measures.

---

## 11. Updates and Versioning

The author may release updated versions of PayKit with bug fixes, security patches, or new features. You are responsible for monitoring releases and updating your deployment. Running outdated versions with known security vulnerabilities is done at your own risk.

---

## 12. Support

PayKit is open-source software maintained on a best-effort basis. Support is provided through:

- GitHub Issues on the official repository
- The `USAGE_GUIDE.md` included with the software

There is no guaranteed response time or SLA for support requests. For priority support, contact the author directly through the repository.

---

## 13. Changes to These Terms

The author reserves the right to update these Terms at any time. Updated terms will be committed to the repository with an updated effective date. Continued use of PayKit after a terms update constitutes acceptance of the revised terms.

---

## 14. Governing Law

These Terms are governed by the laws of Rwanda. Any disputes arising from these Terms or your use of PayKit shall be subject to the jurisdiction of the courts of Kigali, Rwanda.

---

*PayKit is built for East African developers handling real payment infrastructure. Use it responsibly.*