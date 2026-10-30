# Notes on security

Engineering notes on hardening work in chuckd. For reporting a vulnerability
see [SECURITY.md](../SECURITY.md). Open items in the code are marked
`XXX(SECURITY)` and `XXX(DoS)`; `git grep -n 'XXX(SECURITY)\|XXX(DoS)'` lists
them.

## Stack overflow and XDR recursion

- Must use non-recursive quorum set structures across validators, then convert them to recursive ones after entering the validator system. This protects deserialization (and therefore also signature checking) from stack overflow.
