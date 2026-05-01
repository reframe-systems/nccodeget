# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A single-file Go CLI tool that fetches NC Code Tables from an Onshape part studio via the Onshape API and writes them to a folder of text files, one per table instance.

Background: NC Code Tables are custom FeatureScript tables in Onshape that contain CNC toolpath code. Each table has a title like `"14041"` or `"14041, 14042, 14043"` corresponding to the part numbers manufactured by that operation.

## Build and run

```sh
go build         # produces ./nccodeget binary
```

```sh
./nccodeget [-dump] [-settings=<file>] <onshape-part-studio-url> [output-dir]
# e.g.: ./nccodeget https://cad.onshape.com/documents/2ae050.../w/b24063.../e/619427... ./output
# e.g.: ./nccodeget -settings=proxy.json https://cad.onshape.com/... ./output
```

`-dump` writes a raw JSON file of the API response alongside the text files.

The URL must match the pattern `https://cad.onshape.com/documents/{did}/{wvm}/{wvmid}/e/{eid}`.

## Output structure

```
<output-dir>/<element-name>/
  <element-name>.txt         # all tables concatenated
  14040.txt                  # NC code from table titled "14040"
  14041_14042_14043.txt      # commas/spaces in title replaced with _
  <element-name>.json        # raw API response (-dump only)
```

## Credentials

Three settings files (all gitignored) select the target endpoint:

- `direct.json` — direct Onshape API, uses `accessKey`/`secretKey`
- `proxy.json` — production proxy (`https://onshape.reframe.quest`), uses `onshapeKey`/`proxyKey`
- `local.json` — local proxy (`http://localhost:5080`), uses `onshapeKey`/`proxyKey`

`-settings=<file>` defaults to `direct.json`. See `settings.json.template` for all fields.

The pre-commit hook in `hooks/pre-commit` blocks commits that include credential files. To install: `cp hooks/pre-commit .git/hooks/pre-commit`.

## Architecture

Everything lives in `nccodeget.go` (single `main` package). There is no go-client dependency — all API calls use `net/http` directly with basic auth.

**Hardcoded configuration** is in a clearly labeled `const` block at the top of `nccodeget.go`:
- `tableNamespace` — identifies the FeatureScript library that defines `ncCodeTable`. The `m{mid}` component is a microversion ID that may need updating if the library changes.
- `tableType` — `"ncCodeTable"`
- `tableParameters` — `"addPartNumbers=true;addMarkingsFirst=true"`

**Key functions:**
- `main` — parses args/flags, loads credentials, orchestrates the two API calls, writes output files
- `getElementName` — calls `GET /documents/d/{did}/{wvm}/{wvmid}/elements` to resolve the element's human-readable name (used as the output folder and concatenated filename)
- `getFSTable` — calls `GET /partstudios/d/{did}/{wvm}/{wvmid}/e/{eid}/nccodeget` with the hardcoded table config
- `apiGet` — shared HTTP helper used by both API calls
- `extractTableText` — pulls `columnIdToValue["ncCode"]` from each row and joins with newlines
- `sanitizeTitle` — replaces `, ` and `,` with `_` for safe filenames
- `verify` — fatal-error helper used throughout instead of explicit error handling
