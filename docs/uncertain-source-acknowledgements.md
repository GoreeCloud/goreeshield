# GoreeShield: ambiguous source acknowledgements

An error after a local journal submission does not prove the operation failed. Recover by obtaining a fresh authorized source observation for the exact operation ID, tenant, subject and request before deciding whether to retry. An absent observation must remain unresolved; conflicting request details must not be treated as success.

The current Linux journal tests do not inject failures at a real rename or directory sync checkpoint. Do not claim crash safety, external exactly-once effects, signed receipts, or production rights fulfillment from this assessment.
