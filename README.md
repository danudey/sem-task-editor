# sem-task-editor

A TUI for editing your Semaphore CI project's tasks.

## Usage

1. Build it with `go build`
2. Install the [Semaphore CLI](https://github.com/semaphoreci/cli) if you don't have it
3. Run `sem edit project <projectname>` with the `EDITOR` environment variable set to `/path/to/sem-task-editor`; for example:

```sh
EDITOR=sem-task-editor sem edit project myproj
```

This works because the `sem` CLI simply calls `${EDITOR} <path-to-file>`. You can also do this manually to test editing.

## Testing it

If you want to test the program first, you can export the project yaml to a file first and then operate on that:

```sh
sem get project myproj > myproj.yaml
sem-task-editor myproj.yaml
```