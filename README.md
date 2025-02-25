# **kubectl-apidocs**

_A `kubectl` plugin for learning about API resource documentation in a tree view format._

[![License](https://img.shields.io/github/license/hashmap-kz/kubectl-apidocs)](https://github.com/hashmap-kz/kubectl-apidocs/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/hashmap-kz/kubectl-apidocs)](https://goreportcard.com/report/github.com/hashmap-kz/kubectl-apidocs)
[![Workflow Status](https://img.shields.io/github/actions/workflow/status/hashmap-kz/kubectl-apidocs/ci.yml?branch=master)](https://github.com/hashmap-kz/kubectl-apidocs/actions/workflows/ci.yml?query=branch:master)
[![GitHub Issues](https://img.shields.io/github/issues/hashmap-kz/kubectl-apidocs)](https://github.com/hashmap-kz/kubectl-apidocs/issues)
[![Go Version](https://img.shields.io/github/go-mod/go-version/hashmap-kz/kubectl-apidocs)](https://github.com/hashmap-kz/kubectl-apidocs/blob/master/go.mod#L3)
[![Latest Release](https://img.shields.io/github/v/release/hashmap-kz/kubectl-apidocs)](https://github.com/hashmap-kz/kubectl-apidocs/releases/latest)

---

## Examples

Here's a **`kubectl apidocs`** demo:
![apidocs demo GIF](assets/apidocs-demo.gif)

---

## **Installation**

### Using `krew`

1. Install the [Krew](https://krew.sigs.k8s.io/docs/user-guide/setup/) plugin manager if you haven’t already.
2. Run the following command:
   ```bash
   kubectl krew install apidocs
   ```
3. Verify installation:
   ```bash
   kubectl apidocs
   ```

### Manual Installation

1. Download the latest binary for your platform from
   the [Releases page](https://github.com/hashmap-kz/kubectl-apidocs/releases).
2. Place the binary in your system's `PATH` (e.g., `/usr/local/bin`).
3. Example installation script for Unix-Based OS:

   ```bash
   (
     set -euo pipefail

     OS="$(uname | tr '[:upper:]' '[:lower:]')"
     ARCH="$(uname -m | sed -e 's/x86_64/amd64/' -e 's/\(arm\)\(64\)\?.*/\1\2/' -e 's/aarch64$/arm64/')"
     TAG="$(curl -s https://api.github.com/repos/hashmap-kz/kubectl-apidocs/releases/latest | jq -r .tag_name)"

     curl -L "https://github.com/hashmap-kz/kubectl-apidocs/releases/download/${TAG}/kubectl-apidocs_${TAG}_${OS}_${ARCH}.tar.gz" |
       tar -xzf - -C /usr/local/bin && chmod +x /usr/local/bin/kubectl-apidocs
   )
   ```

4. Verify installation:
   ```bash
   kubectl apidocs
   ```

---

## Terminal Navigation Guide

### 🖥️ **Keyboard Shortcuts**

| **Shortcut**   | **Action**                                                           |
| -------------- | -------------------------------------------------------------------- |
| **`<hjkl>`**   | Navigate (Vim-style)                                                 |
| **`<ARROWS>`** | Navigate (Arrow keys)                                                |
| **`<ENTER>`**  | Select (group/resource)                                              |
| **`<TAB>`**    | Switch focus between tree/details (NOTE: details-view is scrollable) |
| **`<ESC>`**    | Step back in navigation                                              |
| **`/`**        | Open search mode                                                     |
| **`<:cmd>`**   | Execute a command                                                    |
| **`<ctrl-c>`** | Quit application                                                     |

---

### 🚀 **Tips for Efficient Navigation**

- **Use `hjkl` for fast movement** (Vim-style navigation).
- **`TAB` lets you quickly switch between tree-view and details (NOTE: details-view is scrollable)**.

---

## **Contributing**

We welcome contributions! To contribute: see the [Contribution](CONTRIBUTING.md) guidelines.

---

## **License**

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

---

## **Additional Resources**

For more information, visit the [project repository](https://github.com/hashmap-kz/kubectl-apidocs).

---
