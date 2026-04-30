# Dashboard visual design PRD

## Problem Statement

`ma`'s dashboard is functionally useful, but it does not yet communicate the product's identity or make the information feel calm, legible, and intentionally arranged. A user can see totals, recent runs, and historical trends, but the current experience feels closer to raw output than to a clear companion surface that helps them quickly understand what `ma` is doing and why it is valuable.

Because the product is named `ma` and already frames itself around gap, space, and pause, the dashboard is missing an important opportunity: it does not yet use negative space, restraint, and rhythm as part of the product experience. The result is a dashboard that can answer questions, but does not yet feel trustworthy, elegant, or memorable enough for regular use or demos.

## Solution

Refine the dashboard into a balanced companion that feels live without becoming noisy. The landing page should prioritize what is happening now, while still showing the long-term value `ma` creates through bytes, words, and token savings. The visual design should express `ma` through spacing, typography, restrained color, and lightweight comparison patterns rather than dense cards, heavy chart chrome, or a control-room aesthetic.

The redesigned dashboard should remain local-only, offline, server-rendered, and additive to the CLI. It should preserve the current product boundaries while making the interface feel edited, intentional, and easier to scan: recent activity first, value second, deeper analysis on a separate stats view, and richer run inspection on dedicated detail pages.

## User Stories

1. As a local `ma` user, I want the dashboard to feel calm and readable, so that I can understand what is happening without visual clutter.
2. As a local `ma` user, I want the landing page to show recent runs first, so that I can immediately tell what `ma` is doing now.
3. As a local `ma` user, I want aggregate savings to remain visible but secondary to live activity, so that I can understand value without losing the sense of current motion.
4. As a local `ma` user, I want recent runs presented in a compact, scannable layout, so that I can compare commands, statuses, and summaries quickly.
5. As a local `ma` user, I want failed runs to stand out clearly, so that problems are visible without the whole interface feeling loud.
6. As a local `ma` user, I want successful runs to stay visually quiet, so that normal activity does not compete with important signals.
7. As a local `ma` user, I want the dashboard's spacing and typography to reflect the idea of `ma`, so that the product feels coherent with its name and purpose.
8. As a local `ma` user, I want the stats page to hold deeper trends and rankings, so that the homepage can stay focused and uncluttered.
9. As a local `ma` user, I want small trend indicators to provide context without large charts taking over the page, so that the dashboard stays compact and calm.
10. As a local `ma` user, I want run detail pages to be easy to read on narrow and wide screens, so that input/output comparisons remain useful in different layouts.
11. As a privacy-conscious user, I want redacted or withheld content to look intentional rather than broken, so that privacy protections feel integrated into the design.
12. As someone demoing `ma`, I want the dashboard to look polished and distinctive, so that it communicates value and product identity immediately.
13. As a maintainer, I want the design system to work with the current local dashboard architecture, so that the visual refinement does not require a new product surface.
14. As a maintainer, I want the visual hierarchy to guide attention with layout, type, and restrained color instead of extra widgets, so that the interface stays maintainable and consistent.
15. As a maintainer, I want any special-purpose display typography to be optional and narrowly scoped, so that distinctive visual touches do not reduce readability or complicate the whole UI.

## Implementation Decisions

- Keep the dashboard as a server-rendered, offline companion surface with lightweight live refresh behavior rather than introducing a separate front-end application.
- Set the homepage's primary role as a balanced live companion: recent runs and current activity receive the strongest emphasis, while aggregate value remains visible in a smaller supporting band.
- Present recent activity as a compact table or list-table hybrid optimized for scanning and comparison, with aligned fields for command, status, time, and short summary.
- Use a restrained visual system built around spacing, typographic hierarchy, subtle dividers, and low-chroma neutrals. Reserve stronger emphasis for active and failed states rather than treating every status as equally loud.
- Keep the value summary compact, with bytes, words, and tokens grouped together and approximate tokens slightly leading as the clearest expression of `ma`'s context-saving value.
- Preserve progressive disclosure across views: the homepage shows live activity and a light teaser of value trends, the stats view carries deeper historical analysis, and run detail pages carry before-and-after inspection.
- Design run detail pages as stacked comparisons by default, with the ability to expand into side-by-side comparison on wider screens to improve readability without forcing cramped layouts.
- Treat redaction, empty states, and unavailable details as deliberate interface states with clear copy and composed spacing rather than placeholder-looking gaps.
- If specialized chart typography is introduced, limit it to microcharts and inline trend indicators inside metrics or tables; the primary reading typography should remain neutral and highly legible.
- Favor small trend indicators and embedded comparison cues over large standalone charts so that the interface stays text-first and consistent with the `ma` concept.

## Testing Decisions

- Validate that the redesigned homepage still lets users identify active, failed, and completed runs quickly without relying on dense visual treatment.
- Validate that recent-run rows remain easy to scan and compare across different data volumes and viewport widths.
- Validate that the summary band still communicates bytes, words, and token savings clearly even though it is secondary to live activity.
- Validate that the stats view carries deeper trend and command-usage analysis without forcing the homepage to absorb the same density.
- Validate that detail pages remain readable in both stacked and wider comparison layouts, including long input/output content.
- Validate that redacted, empty, and no-history states appear intentional and understandable rather than broken or unfinished.
- Validate that the restrained color system preserves enough distinction for active and failed states while keeping success visually quiet.
- Validate that any microchart treatment improves comprehension without reducing accessibility, legibility, or the dashboard's offline simplicity.
- Closest prior art in the repository comes from the current dashboard's overview, stats, and detail split; the existing command-result and metrics vocabulary; and the product's established emphasis on deterministic, local-only behavior.

## Out of Scope

- Reworking the dashboard's core functional behavior, event model, or persistence model beyond what is necessary to support the visual redesign.
- Turning the homepage into a dense multi-panel monitoring console.
- Introducing a separate client-side application architecture or remote design dependency.
- Expanding the visual system into a full branding exercise across the entire CLI.
- Using decorative large-scale charts or animation-heavy interactions as the primary means of communication.
- Replacing the current privacy and sensitive-content posture with more permissive display rules.

## Further Notes

- This PRD is a design-direction companion to the broader dashboard feature PRD rather than a replacement for it.
- The working conclusion from the research is a **balanced companion with light live emphasis**: recent runs first, compact metric trio second, deeper trends on a separate stats page.
- Datatype is in scope only as an optional accent for inline microcharts and small trend indicators, not as the main interface typeface.
- The repository currently has no verified issue-template or triage conventions, so this document is prepared as a markdown artifact in `prd/`.
