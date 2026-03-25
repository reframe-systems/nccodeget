# nccodeget

Fetches NC Code Tables from an Onshape part studio and writes them to a folder of text files, one per table. NC Code Tables are FeatureScript tables containing CNC toolpath code — each table has a title like `"14041"` or `"14041, 14042, 14043"` corresponding to the part numbers manufactured by that operation.

## Prerequisites

You need Go installed. If you don't have it:

1. Go to https://go.dev/dl/ and download the installer for your platform (Mac, Windows, or Linux).
2. Run the installer and follow the prompts.
3. Open a new terminal and verify it worked: `go version`

You should see something like `go version go1.24.3 darwin/arm64`. That's it — no other dependencies are needed.

## Setup

### 1. Get Onshape API credentials

The tool authenticates with Onshape using API keys stored in a `secrets.json` file in this directory. This file is gitignored and never committed.

The credentials live in 1Password. From the `secrets.json.template` file, the paths are:

- **Access key:** `op://Shared/onshape_readonly_api_credentials/username`
- **Secret key:** `op://Shared/onshape_readonly_api_credentials/credential`

Create `secrets.json` in this directory with those values filled in:

```json
{
  "accessKey": "<access key from 1Password>",
  "secretKey": "<secret key from 1Password>"
}
```

### 2. Install the pre-commit hook (optional but recommended)

This prevents you from accidentally committing your credentials:

```sh
cp hooks/pre-commit .git/hooks/pre-commit
```

## Build

From this directory, run:

```sh
go build
```

This compiles the code and produces an executable file called `nccodeget` (or `nccodeget.exe` on Windows) in the same directory. You only need to do this once, or again any time the source code changes.

## Run

```sh
./nccodeget <onshape-part-studio-url> [output-dir]
```

The URL is the address of the part studio in your browser — it should look like:

```
https://cad.onshape.com/documents/2ae050.../w/b24063.../e/619427...
```

`output-dir` is optional and defaults to the current directory.

**Example:**

```sh
./nccodeget "https://cad.onshape.com/documents/2ae050.../w/b24063.../e/619427..." ./output
```

**To also save the raw API response as JSON** (useful for debugging), add `--dump`:

```sh
./nccodeget --dump "https://cad.onshape.com/documents/..." ./output
```

## Output

The tool creates a subdirectory named after the part studio element:

```
output/
  My Part Studio/
    My Part Studio.txt      # all tables concatenated into one file
    14040.txt               # NC code from the table titled "14040"
    14041_14042_14043.txt   # commas and spaces in titles become underscores
    My Part Studio.json     # raw API response (--dump only)
```
