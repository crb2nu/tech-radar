# tech-radar Roadmap

> Last Updated: 2026-07-02
> Tier: 2 (see workspace AGENTS.md "Portfolio Tiers")
> Tracking Issue: none (not in the active-tracking set; backlog linked below)

## Current Status

Deployed tech-radar service (app + postgres + meilisearch) that ingests
dependency/stack scans from workspace repos; this repo also serves
`/ci/radar.yml`, the scanner include consumed by most workspace pipelines —
treat it as a shared contract. Last feature run 2026-05-06 (livez/healthz
probes, Prometheus ingest metrics, drift webhook, schema-on-init fix);
2026-05-27 CI proxy fixes; onboarded to the platform/gitops go CI template
2026-07-02. KNOWN BROKEN: the `tech-radar-gitlab-sync` CronJob is failing
(latest job Failed with 2 Error pods) — tracked as #1 (P2), in Now.

Evidence: last 20 default-branch commits (window 2026-04-16 → 2026-07-02);
Flux Kustomization `tech-radar` Ready; `kubectl get deploy,cronjob -n
tech-radar` → tech-radar 1/1, postgres 1/1, meilisearch 1/1,
tech-radar-gitlab-sync CronJob (17 */6); failing-CronJob finding + pipeline
success 2026-05-27 per .loom/62-functional-health-baseline-2026-07-02.md.

- **Plan store**: plan-workspace-portfolio-refresh-2026-h2-roadmaps-quality-baselin-f3db23
- **Deployed**: k3s `tech-radar` namespace via Flux Kustomization `tech-radar`
- **CI**: platform/gitops go template; also publishes `/ci/radar.yml` consumed workspace-wide

## Now

- [ ] Fix the failing `tech-radar-gitlab-sync` CronJob (#1, P2 — from the
  2026-07-02 functional-health baseline)

## Next

- [ ] Add alerting on gitlab-sync CronJob failures so drift is caught before
  a baseline sweep (file issue when scoped; #1 is the umbrella)

## Later

- Radar visualizer iterations (project lens / ecosystem clustering follow-ups)

## Backlog

Full backlog: [P1 issues](https://gitlab.flexinfer.ai/services/tech-radar/-/issues?label_name[]=P1) ·
[P2](https://gitlab.flexinfer.ai/services/tech-radar/-/issues?label_name[]=P2) ·
[P3](https://gitlab.flexinfer.ai/services/tech-radar/-/issues?label_name[]=P3) ·
[Milestones](https://gitlab.flexinfer.ai/services/tech-radar/-/milestones)
