# Extensions

Extend the CLI with custom executables. Extensions are binaries named `strait-<name>` in your PATH.

## Discover extensions

```bash
strait extension list
```

## Install

```bash
strait extension install github.com/user/strait-myext
```

## Run

```bash
strait extension run myext --flag value
```

## Create a new extension

Scaffold a new extension project:

```bash
strait extension create my-extension
```

## Remove

```bash
strait extension remove myext
```
