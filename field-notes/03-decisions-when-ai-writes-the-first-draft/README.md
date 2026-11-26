# Field note 03 · Making Decisions When AI Writes the First Draft

> **About field notes.** Unlike the labs in this repo, field notes usually describe production work. This one is different: it comes from a three-hour take-home technical assessment for a DevOps / platform engineering role where AI tools were permitted, followed by an AI-free technical review. Every identifying detail is anonymized, and the assessment tasks themselves are omitted; the focus here is why I use AI at all, and how I decided what to accept, rewrite, or reject when an AI assistant wrote the first draft.

| | |
|---|---|
| **Context** | Three-hour take-home technical assessment covering infrastructure review, troubleshooting a local stack, incident analysis, and cost and ownership scenarios. Every important AI suggestion I modified or rejected had to be declared |
| **AI tool** | Claude Code, as an assistant in the terminal |
| **What I delegated** | Repetitive work, and long tasks that are not repetitive but take time to produce |
| **What I kept** | Every decision, and the responsibility for it |

## Why I use AI at all: it is a trade-off

In interviews I have heard many reasons for not using AI: it is not secure; the provider is a US company, far away, and suing it would cost more than the leak itself; the environmental cost. Some of these are fair. A provider such as Anthropic is an external third party and cannot be held accountable for each prompt sent to it or for any credential that ends up in one. Whatever I send leaves my control, and the accountability for it stays with me. Cost is a fair reason too, and a growing one as AI tools become more expensive.

Whatever the reason, the consequence is the same. Nobody pits a cow against a tractor to plow a rice field and expects the cow to keep up. Not buying a tractor can be a sound decision. But then the plan has to be built around the pace of cows, not around the pace of the neighbor who plows with a tractor.

| | Pros | Cons |
|---|---|---|
| **Working with AI** | Analysis, documents, architecture and task execution take far less time | Prompts go to an external provider that is not accountable for them. Drafts can be confidently wrong (rows 5 and 7 below), so every one has to be checked |
| **Working without AI** | Nothing leaves my environment, and every line is mine from the start | The same analysis, documents, architecture and execution take much longer |

So it comes back to the trade-off. I can work without AI at all. But the time I need for analysis, for writing documents, for designing architecture and for executing tasks will not be as short as when I use it. I choose the tractor, and choosing it means managing its risks rather than pretending they are not there: the decisions stay with me, and its output is checked before anything depends on it.

## What I delegate, and what I keep

AI takes the work, not the decisions. What I hand over falls into two kinds of tasks.

- **Repetitive work.** Running the same kinds of checks again and again while troubleshooting: `curl` against each endpoint, container logs, port checks, Prometheus queries, and `terraform validate`, `trivy config` and `trivy image` before and after each change. It also applied the fixes once I had decided on them.
- **Long tasks that are not repetitive.** Turning my review findings into a prioritized table, writing first drafts of the incident analysis and of the cost and ownership answers, looking up external facts (the contents of a GitHub Actions runner image, current ECS deployment features, a supply-chain advisory for a security scanner's GitHub Action, a rotated apt repository key), and explaining AWS concepts I had not mastered by mapping them onto tools I already use: Kubernetes, Argo Rollouts and Loki.

What stayed with me:

- **The order of investigation.** I chose to diagnose from the outside in, and decided which layer to look at next.
- **What to change, and what to leave as a recommendation.** Not every valid improvement belongs in the change set.
- **Trade-offs, and who owns them.** Cost and availability decisions belong to the budget and SLA owner; my job is to prepare them well enough that the owner only has to approve or reject.
- **What counts as proven.** What I stated as fact had a command, a measurement or an official source behind it; what I could not prove was marked as such.

Delegating the work does not delegate the responsibility. Every AI draft is a proposal, and I decide whether it stands.

## Evidence: where my decision differed from the AI's first proposal

The assessment required a declaration of every important AI suggestion I modified or rejected. This is that table, translated from my submitted notes. Each row is a point where the AI's first proposal and my final decision were different.

| # | AI's first proposal | What I decided | Why |
|---|---|---|---|
| 1 | Dev environments running 24/7 and the cross-AZ HA trade-off presented as a list of options, with cross-AZ cost written off as "accepted as an HA premium" | Rewritten as a recommendation, alternatives and a specific approval request, with the cost decision placed with the owner of the budget and the SLA | A list of options without a recommendation does not read as a solution. Cost and availability decisions are not an engineer's to make |
| 2 | Cost increase of 70%: "find the service driving the increase, don't guess" | Intuition still decides which services to check first, then data proves or disproves it | That is how I investigate: intuition speeds things up, data makes sure |
| 3 | Break-glass access mentioned only as an emergency shortcut | Added a trade-off table, what to record on every use, and break-glass frequency as a signal for the SOP owner | Break-glass that keeps being used means the normal workflow does not fit what people actually need |
| 4 | The reason for recording the dev-environment decision: protecting myself | Changed to traceability and the needs of other teams | A decision record is for whoever reviews or is affected by the change later, for example a team that finds its environment stopped after hours and needs to know why and how to start it |
| 5 | Preventive control after the incident: blue/green deployments with CodeDeploy | Replaced with the canary/linear strategies built into ECS | I questioned CodeDeploy. After checking, ECS has had these strategies built in since October 2025, so CodeDeploy was not needed |
| 6 | Remote state and image tag recommendations written generically | Adapted to the patterns I use: a separate bootstrap stack with `use_lockfile`, and `<branch>-<short_sha>` image tags | I only recommend what I can take responsibility for: patterns from my own experience, improved by trial and error and proven by their results |
| 7 | Move to a distroless image, with a Dockerfile comment saying "node as PID 1 receives SIGTERM directly" | Tested, and proven wrong: the change caused a regression, stopping took 10.6 s and ended in SIGKILL (exit 137), while the original image stopped in 0.6 s. I corrected the comment, added a SIGTERM handler, and measured the original image as a baseline | A change that looks like a security fix still has to be tested for runtime behavior |
| 8 | Distroless runtime based on `debian12` | Switched to `debian13` | `trivy image` (Trivy 0.76.2, HIGH and CRITICAL) showed 1 CRITICAL and 6 HIGH in `libssl3` on debian12, and 0 on debian13. Zero only means no CVE **known** to Trivy's database at scan time, not a guarantee: zero-days, CVEs not yet in the database, and findings below HIGH are not covered |
| 9 | Distroless Dockerfile without `USER`, relying on the `:nonroot` tag | Added an explicit `USER nonroot` | Trivy flags it as `DS-0002`. Technically a false positive, but the intent is clearer when it is written down |
| 10 | Close `/metrics` on the public Nginx | Not applied; kept as a recommendation | The assessment listed `GET /metrics` as expected behavior |

## What the final proposals looked like

Rows 1 and 3 changed the shape of a proposal, not just a detail. My proposals always lead with one recommendation and keep the other options next to it as alternatives, so whoever owns the decision can see the trade-offs and still choose differently. Three of them, as submitted:

**Dev environments running 24/7, with nobody asking me to fix it.** I did not wait for a ticket, but I did not shut anything down on my own either. The decision belongs to the lead; the analysis and preparation are mine, so the lead only has to approve or reject.

| | Option | Saving | Risk / effort |
|---|---|---|---|
| **Recommended** | Automatic stop/start schedule outside office hours (ECS desired count to 0, RDS stopped), with a manual way to start it for anyone working late | About 70% of the compute components | Low effort, reversible at any time. A stopped RDS instance starts again after 7 days, so the schedule has to keep stopping it |
| Alternative 1 | Destroy and rebuild daily with Terraform | The largest, including NAT and ALB | High effort, dev data is lost unless re-seeded, slower start every morning |
| Alternative 2 | Only downsize the resources | Smaller | Disrupts nobody, but the saving is limited |

The approval I asked for: a two-week pilot of the recommendation on one environment, with an opt-out for teams that object, followed by a report on the actual saving and any complaints, so the lead can decide whether to extend it.

**Cross-AZ high availability.**

| Decision to approve | Option | Consequence |
|---|---|---|
| Task and DB placement | **Recommended**: AZ affinity, 2 active AZs, RDS Multi-AZ | No app-layer downtime when one AZ fails. Cross-AZ cost from tasks to the DB writer |
| | Alternative: one active AZ, the other on standby | Cross-AZ cost close to zero. Downtime during failover |
| NAT Gateway | **Recommended**: one NAT per AZ | NAT hourly cost grows with the number of AZs. No cross-AZ cost for outbound traffic, and egress survives one AZ failing |
| | Alternative: one shared NAT | Cheaper per hour. Cross-AZ and NAT processing costs, and all egress depends on one AZ |

Each option came with a monthly cost estimate from actual traffic (Cost Explorer and VPC Flow Logs) and the SLA in force. The decision and its reasons go into a ticket or ADR, so it can be revisited when traffic or the SLA changes.

**A developer asking for broad production permissions because a deploy is blocked.**

| | How | When | Trade-off |
|---|---|---|---|
| **Recommended** | Add the specific missing permission to the deploy (CI) role through a reviewed PR to the IAM code | The default, including when the deploy can wait a few hours | Review takes time, but the problem is fixed for good and the next deploy is not blocked again |
| Alternative 1 | Someone already authorized runs the deploy step, with the developer alongside | Urgent, and the step is simple | Fast without widening anyone's permissions, but that person becomes a bottleneck and the gap in the deploy role stays open |
| Alternative 2 | Break-glass: a separate emergency role with approval, MFA, a short session (for example 1 hour) and every action logged in CloudTrail | A production incident or critical hotfix, when the two above are not fast enough | Fastest in an emergency, but the permissions are broad, so one mistake has a large impact. Every use needs a review, and if it is used often the control loses its meaning |

Every break-glass session is recorded: who asked and who approved, why, when it started and ended, what was run, and the follow-up. If break-glass keeps being used, I take that pattern to the owner of the SOP as a proposal to fix the workflow. Changing the SOP is their decision.

## How I checked before deciding

- **Commands over assumptions.** Every change in the lab was verified with a command: HTTP codes with `curl` from the host, `docker compose ps`, Prometheus queries, container stop times and exit codes, `RestartCount` after killing a process, and `trivy image` and `trivy config` before and after.
- **External facts from official sources.** Runner image contents from the `actions/runner-images` readme, ECS features from AWS documentation and announcements, the scanner incident from the official advisory. The rotated apt key was verified with `gpgv` against the repository's `InRelease` file before it was installed.
- **What was not proven is marked.** My hypothesis for the incident's latency, graceful shutdown with a long-running request, and restart after a host reboot were all flagged as untested in the submission.
- **Not applying what should not be applied.** `terraform validate` passing only says the configuration is well-formed. I deliberately did not apply the Terraform to AWS, because applying the code under review would have created the very security problems I was reporting.

## What I take from it

- **It is a trade-off, not a contest.** Refusing AI is a valid choice with real reasons behind it, but then none of its benefits come with that choice either: not the speed, not the repetitive checks it runs, not the long drafts it writes, and not the research and explanations that help me learn faster.
- **Delegate the work, not the responsibility.** Repetitive checks and long drafts are where AI saves the most time. Deciding what they mean, and what to do about them, is still my job.
- **An AI draft is a proposal.** It is reviewed like any other proposal: questioned, tested, and changed when it does not hold.
- **Recommend only what I can take responsibility for.** My patterns come from experience and keep improving through trial and error; what I recommend is the version whose results I have seen.
- **A tool's result proves only what that tool checks.** A passing `terraform validate`, zero HIGH findings, or an HTTP 200 from `/health` each mean less than they appear to.
