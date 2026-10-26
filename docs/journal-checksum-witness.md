# GoreeShield — Local Journal Checksum Witness

Date: October 25, 2026
Status: Linux synthetic-fixture consistency test; not production-ready.

The private witness stores a SHA-256 digest of the published journal bytes. The writer updates the witness only after syncing the journal. Reads reject a mismatched journal with `ErrJournalRollback` or an invalid witness with `ErrJournalWitness`.

The witness is not cryptographic authentication. It shares the same Linux UID with the journal, and both files can be replaced together by the same actor. A missing witness remains accepted for compatibility with older journals. Neither physical power-loss recovery nor genuine external-source effects are proved.
