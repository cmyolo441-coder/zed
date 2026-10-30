#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Builds the SINGLE canonical document `plan.md`.

Sources merged into one file:
  * research paper        (research/paper/*.md)      -> PART I, III, IV, appendix
  * implementation plan   (research/plan_src/*.md)   -> PART II, V, VI, VIII
  * generated file spec   (manifest + spec data)     -> PART VII (308 files)

Section numbers are re-assigned so the merged doc reads as one document.

Run: python3 research/merge_plan.py
"""
import os
import re
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
sys.path.insert(0, HERE)
sys.path.insert(0, os.path.join(HERE, "manifest_data"))
sys.path.insert(0, os.path.join(HERE, "spec_data"))

import plan_build as pb  # noqa: E402

P = os.path.join(HERE, "paper")
S = os.path.join(HERE, "plan_src")


def read(path):
    with open(path, encoding="utf-8") as f:
        return f.read().strip()


def renumber_heads(text, old, new):
    return re.sub(r"^(#{2,4}) %s\." % re.escape(str(old)), lambda m: f"{m.group(1)} {new}.", text, flags=re.M)


def renumber_refs(text, old, new):
    return re.sub(r"§%s(?=[.\s)])" % re.escape(str(old)), f"§{new}", text)


def whole(path, old, new):
    return renumber_refs(renumber_heads(read(path), old, new), old, new)


def grab(path, heading, target, old_top=None, level="###"):
    """Extract one subsection (`### x.y` or `## x`) and re-emit it as `target`."""
    text = read(path)
    m = re.search(r"^%s %s\s+.*$" % (re.escape(level), re.escape(heading)), text, flags=re.M)
    if not m:
        raise SystemExit(f"heading {heading} not found in {path}")
    rest = text[m.end():]
    nxt = re.search(r"^(##|###) ", rest, flags=re.M)
    block = text[m.start():m.end() + (nxt.start() if nxt else len(rest))].strip()
    title = re.sub(r"^#+\s+[\d.]+\s+", "", block.splitlines()[0]).strip()
    body = "\n".join(block.splitlines()[1:]).strip()
    if old_top is not None:
        body = renumber_refs(body, old_top, str(target).split(".")[0])
    hashes = "###" if "." in str(target) else "##"
    return f"{hashes} {target} {title}\n\n{body}".strip()



SLICING = """### 9.20 Agar team chhoti hai — slicing guide

| Team size | Kya banao | Files | LOC | Timeline |
|---|---|---:|---:|---|
| 1–2 devs | **Tier B**: P0 only + `read/write/edit/grep/shell` + 1 provider + basic TUI | 129 | 33k | 6–9 mahine |
| 3–4 devs | P0 + high-value P1 (`compaction`, `mcp`, `state/resume`, `policy/audit`, `orchestrator/dag`) | ~190 | ~49k | 12–15 mahine |
| 5–8 devs | Full P0–P2 | 294 | 73k | 12–14 mahine |
| 8+ devs | P0–P3 + multi-surface | 308 | 76.5k | 13–15 mahine |

Rule of thumb: **P0 se zyada mat kaato.** P0 me hi ~43% code hai kyunki wahi "walking skeleton" hai — bina uske baaki sab hypothetical hai.

**File-level ordering:** `research/file-manifest.csv` ko phase se filter karo; P0 = 129 rows. Har row ka detailed spec Part VII me hai."""

DIV = "\n\n---\n\n"
parts = []

# ------------------------------------------------------------- title block
parts.append(read(os.path.join(S, "00-header.md")))

# ------------------------------------------------------------- PART I
parts.append("# PART I — Research: market, evidence, lessons\n")
front = read(os.path.join(P, "00-front.md"))
front = front.replace("# Enterprise-Grade Terminal AI Agent in Rust\n"
                      "## Deep Research Blueprint — File Count, Architecture, Effort aur Build Plan\n", "")
front = re.sub(r"^\*\*(Document type|Target|Date|Prepared for|Companion artifacts|Machine-readable data):\*\*.*$", "", front, flags=re.M)
front = re.sub(r"\n{3,}", "\n\n", front).strip().lstrip("-").strip()
parts.append(front)

# ------------------------------------------------------------- PART II
parts.append("# PART II — Product: mission, scope, surface\n")
parts.append(whole(os.path.join(S, "01-mission.md"), 1, 2))
parts.append(whole(os.path.join(S, "02-product.md"), 2, 3))

# ------------------------------------------------------------- PART III
parts.append("# PART III — Architecture\n")
parts.append(whole(os.path.join(P, "02-architecture.md"), 2, 4))
parts.append(whole(os.path.join(S, "03-architecture.md"), 3, 5))

# ------------------------------------------------------------- PART IV
parts.append("# PART IV — Subsystems\n")
parts.append(whole(os.path.join(P, "03a-subsystems.md"), 3, 6))
parts.append(whole(os.path.join(P, "03b-subsystems.md"), 3, 6))

# ------------------------------------------------------------- PART V
parts.append("# PART V — Parallel subagent system\n")
parts.append(whole(os.path.join(S, "04-subagents.md"), 4, 7))

# ------------------------------------------------------------- PART VI
parts.append("# PART VI — Delivery: phases, effort, risks, decisions\n")
delivery = whole(os.path.join(S, "05-delivery.md"), 5, 8)
effort = grab(os.path.join(P, "06-execution.md"), "6.1", "8.9", old_top=6)
adr = grab(os.path.join(P, "06-execution.md"), "6.5", "8.10", old_top=6)
deps = grab(os.path.join(P, "06-execution.md"), "6.6", "8.11", old_top=6)
migr = grab(os.path.join(P, "06-execution.md"), "6.8", "8.12", old_top=6)
parts.append("\n\n".join([delivery, effort, adr, deps, migr]))

# ------------------------------------------------------------- PART VII
parts.append("# PART VII — File-by-file specification (308 files)\n")
pb.check()
spec_md, n_files, n_loc = pb.file_spec_markdown(section=9)
parts.append(spec_md + "\n" + SLICING)

# ------------------------------------------------------------- PART VIII
parts.append("# PART VIII — Contracts, quality, enterprise, execution\n")
parts.append(whole(os.path.join(S, "07-contracts.md"), 7, 10))

quality = whole(os.path.join(S, "08-quality.md"), 8, 11)
dist = grab(os.path.join(P, "04-engineering.md"), "4.4", "11.8", old_top=4)
xplat = grab(os.path.join(P, "04-engineering.md"), "4.5", "11.9", old_top=4)
docs = grab(os.path.join(P, "04-engineering.md"), "4.6", "11.10", old_top=4)
obs = grab(os.path.join(P, "04-engineering.md"), "4.7", "11.11", old_top=4)
parts.append("\n\n".join([quality, dist, xplat, docs, obs]))

parts.append(whole(os.path.join(S, "09-enterprise.md"), 9, 12))
parts.append(whole(os.path.join(S, "10-execution.md"), 10, 13))

# ------------------------------------------------------------- PART IX
parts.append("# PART IX — Appendix\n")
src_block = grab(os.path.join(P, "06-execution.md"), "7. Sources", "14", level="##")
src_block = src_block.replace("## 14 Sources", "## 14. Sources")
parts.append(src_block)

doc = DIV.join(p.strip() for p in parts if p and p.strip())
doc = re.sub(r"\n{3,}", "\n\n", doc)
doc = re.sub(r"\n{3,}---", "\n\n---", doc)
doc = doc.replace("\n\n---\n\n---\n\n", "\n\n---\n\n")
doc = re.sub(r"\n---\n\n(# PART [^\n]+\n)\n---\n\n", r"\n\n\1\n", doc)
doc = re.sub(r"\n---\n\n(?=# PART )", "\n\n", doc)
doc = re.sub(r"(?<!\n---)\n\n(?=# PART )", "\n\n---\n\n", doc)
doc = re.sub(r"\n{3,}---\n", "\n\n---\n", doc)
doc = re.sub(r"\n{3,}(# PART )", r"\n\n\1", doc)

# ------------------------------------------------- cross-reference fixups
REF_FIXES = [
    ("(§4)", "(§7)"),
    ("(§8)", "(§11)"),
    ("eval harness §8", "eval harness §11.3"),
    ("§4.15", "§7.15"),
    ("§4.13", "§7.13"),
    ("§3.7.4", "§6.7.4"),
    ("pattern grammar §7.4", "pattern grammar §10.4"),
    ("§9.2 se", "§12.2 se"),
    ("§10.2 DoD", "§13.2 DoD"),
    ("backlog §10 me", "backlog §13 me"),
    ("attribution chain §9", "attribution chain §12"),
    ("Scope freeze §1.4", "Scope freeze §2.4"),
    ("§5 ka P0-only slice", "§9.20 ka slicing guide"),
    ("`research/file-manifest.csv` (blueprint §5 ka sizing table) se aate hain.",
     "`research/file-manifest.csv` se aate hain."),
    ("(detailed table: blueprint §3.7.4)", "(detailed table: §6.7.4)"),
    ("(detailed table: blueprint §6.7.4)", "(detailed table: §6.7.4)"),
    ("**Is paper ka concrete deliverable:** ek 308-file / 76,560-LOC blueprint — har file ka naam, role, estimated LOC aur delivery phase ke saath. Plus effort math, team plan, risk register, ADRs aur dependency list.",
     "**Is document ka concrete deliverable:** ek 308-file / 76,560-LOC blueprint — har file ka naam, role, estimated LOC aur delivery phase ke saath (Part VII), plus effort math, team plan, risk register, ADRs aur dependency list (Part VI)."),
    ("Main is paper me", "Main is document me"),
    ("Ye 12 pillars is blueprint ke acceptance criteria hain", "Ye 12 pillars is plan ke acceptance criteria hain"),
    ("### 4.3 Event-sourced turn (ye blueprint ka sabse important decision)", "### 4.3 Event-sourced turn (sabse important architectural decision)"),
    ("**Maarta hai (aur iska mitigation blueprint me hai):**", "**Maarta hai (aur iska mitigation §4.7 me hai):**"),
    ("Blueprint decisions:", "Design decisions:"),
    ("(policy-gated, P3 in blueprint)", "(policy-gated, P3)"),
    ("(§blueprint 6.8)", "(§8.12)"),
]
for a, b in REF_FIXES:
    doc = doc.replace(a, b)

# doc map + reading paths (header tables) — rewritten for merged numbering
OLD_MAP_START = "## Document map"
NEW_MAP = """## Document map (ek hi file, 9 parts)

| Part | § | Section | Kya milega |
|---|---|---|---|
| **I** | 0–1 | Research | TL;DR + file-count answer, method & measured data, product landscape, 12 enterprise pillars, harness evidence |
| **II** | 2–3 | Product | Mission, personas, scope/non-goals, 12 golden rules, success metrics, commands, modes, config, exit codes, UX flows |
| **III** | 4–5 | Architecture | 7 planes, event-sourced core, turn state machine, concurrency, latency budget, Rust trade-offs, 19-crate map, dependency rules, failure semantics |
| **IV** | 6 | Subsystems | Context economy, tool/patch plane, model gateway, sandbox & permissions, state/resume, TUI, MCP/ACP/hooks/skills, governance, threat model |
| **V** | 7 | **Parallel subagents** | DAG planner, deterministic scheduler, file leases, isolation levels, merge queue + verify gates, safety, metrics |
| **VI** | 8 | Delivery | P0–P3 phases, 15 sprints, gates, ownership, effort math, 15 ADRs, dependency list, Go→Rust migration |
| **VII** | 9 | **File-by-file spec** | **308 files**: exact code + features/acceptance + LOC + phase, crate by crate |
| **VIII** | 10–13 | Contracts → Execution | Event/op/tool catalogues, policy grammar, DB schema, error codes · test matrix, evals, perf budgets, CI · enterprise readiness · DoD, risks, backlog, traceability |
| **IX** | 14 | Appendix | Sources |

## Padhne ka tareeka

| Aap kaun ho | Padho |
|---|---|
| Founder / decision maker | §0, §2, §8 (effort + risks), §13.7 (aap se 8 decisions) |
| Tech lead | Sab; especially §4, §5, §7, §8.10 (ADRs), §11 |
| Implementer (day 1) | §9 (apni crate), §10 (contracts), §11 (tests/DoD), §8.2–8.3 (sprint) |
| Reviewer / architect | §4.1 (golden rules ke saath), §5.2 (forbidden edges), §7, §11.4 (budgets), §12 |
| Security reviewer | §6.4, §6.7.4 (threat model), §12 |
"""
doc = re.sub(r"## Document map.*?(?=## Padhne ka tareeka)", NEW_MAP, doc, flags=re.S)
doc = re.sub(r"## Padhne ka tareeka.*?(?=## Teen non-negotiables)", "", doc, flags=re.S)

out = os.path.join(ROOT, "plan.md")
with open(out, "w", encoding="utf-8") as f:
    f.write(doc + "\n")
print(f"wrote {out}: {n_files} files, {n_loc:,} LOC, {len(doc.splitlines())} lines, ~{len(doc.split()):,} words")
