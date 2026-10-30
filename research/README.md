# Research & planning sources — Rust terminal-agent

**Deliverable is one file: [`../plan.md`](../plan.md)** (~30k words, 2,588 lines, 9 parts, §0–§14).
Everything below is *build input* for that file — you should not need to read any of it.

| File | What it is |
|---|---|
| `../plan.md` | **The single document.** Research → product → architecture → subsystems → parallel subagents → delivery → 308-file spec → contracts → quality → enterprise → execution → sources |
| `file-manifest.csv` | 308 rows: `id, crate, path, purpose, est_loc, phase` (machine-readable sizing) |
| `manifest_data/d1–d6_*.py` | Sizing source of truth (edit here → regenerate CSV + plan) |
| `spec_data/s1–s6_*.py` | Per-file **code + features/acceptance** text used by `plan.md` Part VII |
| `plan_src/*.md` | Hand-written plan sections (Parts II, V, VI, VIII) |
| `paper/*.md` | Research paper sections (Parts I, III, IV, and §8.9–8.12, §11.8–11.11, §14) |
| `build_manifest.py` | Rebuilds `file-manifest.csv` and prints crate totals |
| `plan_build.py` | Library: manifest loading, `check()` coverage assert (308/308), `file_spec_markdown()` |
| `merge_plan.py` | **Assembler** — merges paper + plan + generated spec into `../plan.md`, rewrites §cross-references to the merged numbering |

Regenerate the single document:

```bash
python3 research/build_manifest.py && python3 research/merge_plan.py
```

`merge_plan.py` fails loudly if a spec entry is missing for any manifest row (coverage != 308/308)
or if a paper/plan heading it expects has been renamed.

Numbering in the merged doc: §0–1 research · §2–3 product · §4–5 architecture ·
§6 subsystems · §7 parallel subagents · §8 delivery · §9 file-by-file spec (9.1–9.20) ·
§10 contracts · §11 quality · §12 enterprise · §13 execution · §14 sources.
