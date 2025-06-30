# Contributing

Thanks for your interest in improving `stdx`.

## Guidelines

- **Use the issue tracker.**  
  If you have a question or need help, please open an issue first.
- **Keep it focused.** Avoid multiple unrelated changes in a single PR.
- **Follow the style.** Attempt to match the existing code and comment style.
- **Test it.** Run `go test -race ./...` and make sure it passes.
- **Update the changelog.**  
  Add a new top-level version section to `CHANGELOG.md`.  
  Follow this format — it's used for automated tagging:

  ```markdown
  ## [vX.Y.Z] - YYYY-MM-DD

  Added:
  - New stuff

  Changed:
  - Changed stuff

  ...
  ```

- **Submit the PR.**  
  Make sure your branch is up-to-date with `main` before submitting.  
  I'll review it when I can <3
