#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Builds the file manifest (CSV) for the Rust terminal-agent plan.

Run:  python3 research/build_manifest.py
Out:  research/file-manifest.csv   (+ summary printed to stdout)
"""
import csv
import os
import sys
from collections import Counter, OrderedDict

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, "manifest_data"))

from d1_cli_protocol_core import DATA as D1        # noqa: E402
from d2_model_context_index import DATA as D2      # noqa: E402
from d3_tools_exec_sandbox import DATA as D3       # noqa: E402
from d4_state_tui_server import DATA as D4         # noqa: E402
from d5_policy_auth_ext_misc import DATA as D5     # noqa: E402
from d6_orchestrator import DATA as D6             # noqa: E402

ALL = {}
for d in (D1, D2, D3, D4, D5, D6):
    ALL.update(d)

ORDER = [
    "cli", "protocol", "core", "model", "context", "index", "tools",
    "exec", "sandbox", "policy", "state", "tui", "server", "orchestrator", "ext",
    "auth", "git", "telemetry", "testkit",
]

missing = [k for k in ORDER if k not in ALL]
extra = [k for k in ALL if k not in ORDER]
assert not missing and not extra, (missing, extra)

rows = []
for crate in ORDER:
    meta = ALL[crate]
    for rel, purpose, loc, phase in meta["files"]:
        rows.append({
            "crate": crate,
            "path": f"crates/{crate}/{rel}",
            "purpose": purpose,
            "est_loc": loc,
            "phase": phase,
            "tests_est": 0,
        })

# test files estimate: 0.6 test/snapshot files per hand-written source file (rust tests + insta)
for r in rows:
    r["tests_est"] = 0

by_crate = OrderedDict()
for crate in ORDER:
    cr = [r for r in rows if r["crate"] == crate]
    by_crate[crate] = {
        "desc": ALL[crate]["desc"],
        "files": len(cr),
        "loc": sum(r["est_loc"] for r in cr),
        "tests": round(len(cr) * 0.6),
        "phases": Counter(r["phase"] for r in cr),
    }

total_files = len(rows)
total_loc = sum(r["est_loc"] for r in rows)
total_tests = sum(v["tests"] for v in by_crate.values())


# ---------- CSV ----------
with open(os.path.join(HERE, "file-manifest.csv"), "w", newline="", encoding="utf-8") as f:
    w = csv.DictWriter(f, fieldnames=["id", "crate", "path", "purpose", "est_loc", "phase"])
    w.writeheader()
    for i, r in enumerate(rows, 1):
        w.writerow({"id": i, "crate": r["crate"], "path": r["path"],
                    "purpose": r["purpose"], "est_loc": r["est_loc"], "phase": r["phase"]})

# ---------- per-crate markdown ----------
for crate, info in by_crate.items():
    cr = [r for r in rows if r["crate"] == crate]
    lines = [f"### `crates/{crate}` — {info['files']} files · ~{info['loc']:,} LOC", "",
             f"{info['desc']}", "",
             "| # | File | Role | LOC | Phase |", "|---|---|---|---:|---|"]
    for i, r in enumerate(cr, 1):
        rel = r["path"].split(f"crates/{crate}/", 1)[1]
        lines.append(f"| {i} | `{rel}` | {r['purpose']} | {r['est_loc']} | {r['phase']} |")
    lines.append("")

# ---------- summary ----------
s = ["",
     "| Crate | Files | Est. LOC | Test files (est.) | Phase mix |", "|---|---:|---:|---:|---|"]
for crate, info in by_crate.items():
    mix = " ".join(f"{k}:{v}" for k, v in sorted(info["phases"].items()))
    s.append(f"| `{crate}` | {info['files']} | {info['loc']:,} | {info['tests']} | {mix} |")
s += ["", f"| **Total** | **{total_files}** | **{total_loc:,}** | **{total_tests}** | |", ""]
print("\n".join(s))
print(f"\navg LOC/file: {total_loc/total_files:.0f}")
print(f"grand total tracked files (src+tests+snapshots est): {total_files + total_tests + total_tests*1.4:.0f}")
print(f"p0 files: {sum(1 for r in rows if r['phase']=='P0')}, "
      f"p0 loc: {sum(r['est_loc'] for r in rows if r['phase']=='P0'):,}")
