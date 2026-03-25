# Phase Learnings

Project-specific patterns, gotchas, and solutions.
Searched by `/gsd:discuss-phase` and `/gsd:plan-phase`.

*Last updated: 2026-03-25*

---

### Archived/unmaintained dependencies: study architecture, don't depend on binary
<!-- problem_type: architecture -->
<!-- component: proxy/dependencies -->
<!-- root_cause: Using archived open-source projects (e.g., compy) as runtime dependencies creates fragile builds -->
<!-- resolution_type: design_change -->
<!-- severity: high -->
<!-- date: 2026-03-25 -->

**Problem:** A key dependency (compy, an HTTP proxy library) was archived and unmaintained, causing build failures with newer Go versions and missing security patches.
**Root Cause:** Depending on the compiled binary/library of an unmaintained project means you inherit all its technical debt with no path to fixes. Archived repos don't get security patches, dependency updates, or bug fixes.
**Solution:** Study the architecture and design patterns of the archived project, then implement the needed functionality directly. Extract the ideas, not the code. Fork only as a last resort (you inherit maintenance burden).
**Key Insight:** Archived dependencies are documentation, not dependencies. Read them to understand the approach, then build your own implementation. This gives you full control over the code you ship and eliminates the "dependency graveyard" problem.

### OverlayFS must be enabled AFTER deployment, not during
<!-- problem_type: bug -->
<!-- component: system/filesystem -->
<!-- root_cause: Enabling OverlayFS before deployment completes makes the filesystem read-only, preventing file writes needed during setup -->
<!-- resolution_type: fix -->
<!-- severity: critical -->
<!-- date: 2026-03-25 -->

**Problem:** Deployment script failed because file writes were rejected after enabling OverlayFS partway through the setup process.
**Root Cause:** OverlayFS makes the lower filesystem read-only, redirecting writes to an upper (tmpfs/RAM) layer. If enabled during deployment, all subsequent file writes go to the RAM overlay and are lost on reboot. Worse, if the overlay fills up, writes fail entirely.
**Solution:** Complete all deployment steps (file copies, config writes, permission changes) FIRST, then enable OverlayFS as the final step. The deployment script should have a clear "point of no return" marker.
**Key Insight:** OverlayFS is a boot-time concern, not a deploy-time concern. Deploy to the real filesystem, verify everything works, then enable the overlay for runtime protection. Treat OverlayFS activation like flipping a "lock" switch -- do it last.
