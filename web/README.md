# Anga's Online Compiler

A browser-based compiler for **C, C++, Python, and Java**.

Source code: <https://github.com/Anga205/CodeExecutionAPI>

## How to use

1. Go to the [website](https://compiler.anga.codes/).
2. Pick a language from the **Language** control.
3. Write or paste code in the **Code** tab.
4. If your program reads input, open the **Input (stdin)** tab and add one line per input.
5. Press **Run code** (or `Ctrl`/`Cmd` + `Enter`).
6. Read the program output, time, and memory in the **Output** pane.

## Execution constraints

Code runs in a controlled environment. Limits are shown in the **Limits** control and can be changed there.

- Time limit: 5 seconds by default, 1–60 seconds allowed.
- Memory limit: 32 MB by default, 8–128 MB allowed.

## Development

This site is built with [Astro](https://astro.build/) and uses [bun](https://bun.sh/) as the package manager.

```bash
bun install
bun run dev      # local dev server
bun run build    # production build
bun run preview  # preview the production build
```

The frontend calls the execution API at `https://codeapi.anga.codes/v2/execute`. Override it with the `PUBLIC_EXECUTE_ENDPOINT` environment variable.

