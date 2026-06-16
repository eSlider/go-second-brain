# ADR-0004: Ponytail minimalism

- Status: Accepted
- Context: SDK should stay small and boring; avoid framework churn.
- Decision: Follow [Ponytail lazy senior dev](https://github.com/DietrichGebert/ponytail): YAGNI, stdlib first, no new dependencies without need, deletion over addition. Intentional shortcuts get a `ponytail:` comment naming the ceiling and upgrade path.
- Consequences: Non-trivial logic gets one small runnable check (test or self-check). No abstractions unless explicitly requested.
