# Notes on security

## Stack overflow and XDR recursion

- Must use non-recursive quorum set structures across validators, then convert them to recursive ones after entering the validator system. This protects deserialization (and therefore also signature checking) from stack overflow.
