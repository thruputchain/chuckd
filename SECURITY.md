# Security policy

## Reporting a vulnerability

Please do not report security vulnerabilities through public GitHub issues,
pull requests or discussions.

Report them privately through GitHub's
[private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability)
for this repository: open the **Security** tab and choose **Report a
vulnerability**.

Include what you can of the following:

- the component and version or commit affected
- steps to reproduce, or a proof of concept
- the impact you believe it has

We will acknowledge the report, keep you informed as we work on a fix, and
credit you in the advisory unless you ask us not to.

## Supported versions

Only the most recent release receives security fixes.

## Known hardening work

Places where the code has known gaps that are tracked but not yet closed are
marked `XXX(SECURITY)` and `XXX(DoS)` in the sources, and summarized in
[doc/security-notes.md](doc/security-notes.md). Reports about those areas are
still welcome if they show an impact beyond what the notes describe.
