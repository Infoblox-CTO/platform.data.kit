---
name: dk-validate
description: Run a full validation pass on a dk project or package
user_invocable: true
---

# dk validate — Full Project Validation

Run the complete dk validation workflow against the current project or a specific package.

## Steps

1. **Load the CLI reference** for context on flags and error codes:
   ```bash
   dk docs -o llm
   ```

2. **Check environment prerequisites:**
   ```bash
   dk doctor
   ```

3. **Lint all packages** in the project (strict mode catches warnings too):
   ```bash
   dk lint --scan-dir . --strict
   ```

4. **Validate all dataset manifests** found under the project:
   ```bash
   find . -path '*/dataset/*.yaml' -exec dk dataset validate {} --offline \;
   ```

5. **Show the pipeline dependency graph** to verify connectivity:
   ```bash
   dk pipeline show --scan-dir .
   ```

6. **Dry-run build** to verify artifact packaging without pushing:
   ```bash
   dk build --scan-dir . --dry-run
   ```

7. **Report results** — summarise any errors, warnings, or disconnected nodes in the pipeline graph.

## Error Handling

- If `dk doctor` reports failures, fix those first (missing tools, Docker not running, etc.).
- If `dk lint` reports errors, look up the error code in the output of `dk docs -o llm` under the `errors:` section for the fix guidance.
- If the pipeline graph shows disconnected datasets, check that Transform inputs/outputs reference existing DataSet names.
