#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Builds plan.md: assembles hand-written sections + generated file-by-file spec (§6)."""
import os
import sys
from collections import OrderedDict

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
sys.path.insert(0, os.path.join(HERE, "manifest_data"))
sys.path.insert(0, os.path.join(HERE, "spec_data"))

from d1_cli_protocol_core import DATA as D1            # noqa: E402
from d2_model_context_index import DATA as D2          # noqa: E402
from d3_tools_exec_sandbox import DATA as D3           # noqa: E402
from d4_state_tui_server import DATA as D4             # noqa: E402
from d5_policy_auth_ext_misc import DATA as D5         # noqa: E402
from d6_orchestrator import DATA as D6                 # noqa: E402
from s1_cli_protocol_core import SPEC as S1            # noqa: E402
from s2_model_context_index import SPEC as S2          # noqa: E402
from s3_tools_exec_sandbox import SPEC as S3           # noqa: E402
from s4_policy_state_tui import SPEC as S4             # noqa: E402
from s5_server_orchestrator_ext import SPEC as S5      # noqa: E402
from s6_git_telemetry_testkit import SPEC as S6        # noqa: E402

MANIFEST = {}
for d in (D1, D2, D3, D4, D5, D6):
    MANIFEST.update(d)
SPEC = {}
for s in (S1, S2, S3, S4, S5, S6):
    SPEC.update(s)

# build order for the spec section (dependency-ish, matches §5.5 of the paper)
ORDER = [
    "protocol", "testkit", "model", "core", "policy", "context", "index",
    "tools", "exec", "sandbox", "state", "tui", "cli", "server",
    "orchestrator", "ext", "auth", "git", "telemetry",
]

TITLES = {
    "protocol": "Wire contracts (types only)",
    "testkit": "Test infrastructure",
    "model": "Model gateway — providers, streaming, cost",
    "core": "Harness — turn loop, sessions, approvals, budgets",
    "policy": "Control plane — rules, redaction, audit",
    "context": "Context economy — assembly, compaction, memory",
    "index": "Code intelligence — repo map, search",
    "tools": "Capability surface — file/patch/shell/LSP/web",
    "exec": "Process execution — spawn, PTY, output",
    "sandbox": "OS isolation — bwrap/Landlock/Seatbelt/AppContainer",
    "state": "Persistence — JSONL truth, SQLite index, snapshots",
    "tui": "Terminal UI — render, input, approval UX",
    "cli": "Entrypoints — subcommands, headless, doctor, update",
    "server": "Headless surfaces — JSON-RPC app-server + ACP",
    "orchestrator": "Parallel subagents — DAG, leases, merge gates",
    "ext": "Extensibility — skills, hooks, plugins, commands",
    "auth": "Credentials — keyring, OAuth, CI identity",
    "git": "Repository awareness — status, diff, worktrees, checkpoints",
    "telemetry": "Observability — tracing, OTLP, support bundle",
}


def check():
    errs = []
    for crate in ORDER:
        if crate not in MANIFEST:
            errs.append(f"manifest missing crate {crate}")
            continue
        m = [f[0] for f in MANIFEST[crate]["files"]]
        s = [e[0] for e in SPEC.get(crate, [])]
        if not s:
            errs.append(f"spec missing crate {crate}")
            continue
        for p in m:
            if p not in s:
                errs.append(f"spec missing {crate}/{p}")
        for p in s:
            if p not in m:
                errs.append(f"spec extra {crate}/{p}")
        if len(m) != len(set(m)):
            errs.append(f"duplicate manifest paths in {crate}")
    if errs:
        print("\n".join(errs))
        raise SystemExit(f"{len(errs)} coverage errors")


def esc(t):
    """Escape pipes so they cannot break markdown table cells."""
    return t.replace("|", chr(92) + "|")


def file_spec_markdown(section=6):
    out = []
    out.append(f"## {section}. File-by-file specification — 308 files\n")
    out.append(
        "Har file ke saamne do cheezein hain: **Code** (andar exactly kya likhna hai — structs, functions, "
        "algorithms) aur **Features / acceptance** (kaise pata chalega ki wo file sahi hai). "
        "LOC aur phase Part VI ke delivery phases (P0–P3) ke hisaab se hain.\n")
    out.append("Legend: `P0` walking skeleton · `P1` usable product · `P2` enterprise GA · `P3` differentiation.\n")

    total_f = total_l = 0
    for crate in ORDER:
        files = MANIFEST[crate]["files"]
        spec = {e[0]: e for e in SPEC[crate]}
        loc = sum(f[2] for f in files)
        total_f += len(files)
        total_l += loc
        out.append(f"### {section}.{ORDER.index(crate)+1} `crates/{crate}` — {len(files)} files · ~{loc:,} LOC")
        out.append(f"_{TITLES.get(crate, '')}_ — {MANIFEST[crate]['desc']}\n")
        out.append("| File | LOC · Phase | Code (kya likhna hai) | Features / acceptance |")
        out.append("|---|---|---|---|")
        for rel, _role, floc, phase in files:
            _, code, feat = spec[rel]
            out.append("| `%s` | %s · %s | %s | %s |" % (rel, floc, phase, esc(code), esc(feat)))
        out.append("")
    out.append(f"**Total: {total_f} files · ~{total_l:,} LOC.** "
               f"Test files (§8) isme count nahi hain — unka estimate 0.6 × per source file hai.\n")
    out.append("---\n")
    return "\n".join(out), total_f, total_l


SECTIONS_BEFORE = ["00-header.md", "01-mission.md", "02-product.md", "03-architecture.md", "04-subagents.md", "05-delivery.md"]
SECTIONS_AFTER = ["07-contracts.md", "08-quality.md", "09-enterprise.md", "10-execution.md"]


def main():
    print("plan_build.py is a library now — canonical doc build karo: python3 research/merge_plan.py")
    return
    check()
    body, n, loc = file_spec_markdown()
    parts = []
    for s in SECTIONS_BEFORE:
        with open(os.path.join(HERE, "plan_src", s), encoding="utf-8") as f:
            parts.append(f.read().strip())
    parts.append(body.strip())
    for s in SECTIONS_AFTER:
        with open(os.path.join(HERE, "plan_src", s), encoding="utf-8") as f:
            parts.append(f.read().strip())
    doc = "\n\n".join(parts) + "\n"
    out = os.path.join(ROOT, "plan.md")
    with open(out, "w", encoding="utf-8") as f:
        f.write(doc)
    print(f"wrote {out}: {n} files, {loc:,} LOC, {len(doc.splitlines())} lines, ~{len(doc.split()):,} words")


if __name__ == "__main__":
    main()
