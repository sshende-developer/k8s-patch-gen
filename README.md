## `k8s-patch-gen`

📦 A powerful, interactive CLI tool to help you generate **Kubernetes resource modifier YAMLs** — specifically for [Velero](https://velero.io) restore customization, with plans to support broader Kubernetes patching use cases.

## ✨ Features

- ✅ Interactive prompts to guide YAML creation
- 📜 Refer to an existing manifest file for help choosing patch paths
- 🔍 Supports wildcards like `*.apps` and `*.*` for `groupResource`
- 🎯 Smart JSON path extraction from YAML line numbers
- 🧩 Supports all [JSON Patch RFC6902](https://datatracker.ietf.org/doc/html/rfc6902) operations:
  - `add`, `replace`, `remove`, `copy`, `move`, `test`
- 🧱 Nested JSON builder for complex `add` payloads
- 🎯 Regex builder for `resourceNameRegex` without needing to write regex manually
- 💡 Works with or without existing manifests

---

## 📦 Installation

```bash
git clone https://github.com/<your-org>/k8s-patch-gen.git
cd k8s-patch-gen
make build

```

The CLI binary will be built at:

```bash
./bin/generateK8sPatchfile
```

---

## 🚀 Usage

Run the interactive YAML generator:

```bash
./bin/generateK8sPatchfile generate
```

You’ll be guided step-by-step through:

1. 📄 Whether to reference an existing Kubernetes manifest
2. 🧩 Specifying resource conditions (groupResource, regex, namespaces, etc.)
3. 🛠️ Selecting JSON Patch operations (`add`, `replace`, `remove`, `copy`, etc.)
4. ✏️ Providing patch paths and values interactively
5. 💾 Saving your patch YAML to disk

---

## 📘 Example: What Gets Generated?

Here’s a sample output YAML you can generate with this tool:

```yaml
version: v1
resourceModifierRules:
- conditions:
    groupResource: virtualmachines.kubevirt.io
    resourceNameRegex: ".*"
  patches:
  - operation: replace
    path: "/spec/runStrategy"
    value: "Halted"
  - operation: replace
    path: "/spec/running"
    value: false
```

---

## 🧪 Running Tests

```bash
make test
```

Tests include:

- ✅ `detectGroupResource()` logic
- ✅ YAML line-number based path extraction
- ✅ Smart nested JSON input builder
- ✅ Regex construction from simple patterns

---

## 📂 Project Structure

| File | Description |
|------|-------------|
| `cmd/generate.go` | Main logic for interactive patch file generation |
| `main.go`         | Entry point and CLI wiring |
| `Makefile`        | Build/test helpers |
| `generate_test.go`| Unit tests |

---

## 📄 License

[MIT](LICENSE)

---

## 🙌 Contributing

We welcome issues, ideas, and PRs! If you’ve got suggestions to improve the CLI UX, extend support to other tools like `kubectl patch`, or integrate directly with Velero CRDs — we’re all ears.

---

## 🔗 Related Resources

- [Velero Resource Modifiers Docs](https://velero.io/docs/)
- [RFC6902 (JSON Patch)](https://datatracker.ietf.org/doc/html/rfc6902)

---

## ⭐️ Star this repo if you find it useful!
