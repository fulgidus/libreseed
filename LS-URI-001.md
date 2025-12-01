# Libreseed URI Specification
Specification ID: LS-URI-001  
Version: 1.0.0  
Status: Draft Proposal  
Authors: Libreseed Working Group  
Audience: Implementers of Libreseed clients, package managers, and dependency resolvers.

---

# 1. Introduction

This document specifies the `libreseed://` URI scheme used for identifying and resolving packages distributed through the Libreseed decentralized package network.

Libreseed is a peer-to-peer, BitTorrent-style system that stores software packages as content-addressed chunk trees. Each package is cryptographically tied to a **self-authenticating identity**, enabling authenticity verification without any central keys, registries, or out-of-band trust.

The URI scheme defined in this document is:

- **Decentralized** — no central registry or global namespace.
- **Self-authenticating** — identity derives from the author’s public key.
- **Reproducible** — supports optional root hash pinning.
- **Language-agnostic** — works across Go, Zig, Node (pnpm/Yarn), Python, etc.
- **Flexible** — supports latest-version resolution while remaining secure.

---

# 2. Terminology

| Term | Meaning |
|------|---------|
| **Identity** | A multibase/multihash value derived from the author’s public key. |
| **Package name** | Human-friendly name, e.g. `mypkg`. |
| **Version** | Version string from the package manifest. |
| **Root hash** | Content hash of the package payload. |
| **Manifest** | Metadata file embedded in each package. |
| **Resolver** | Client responsible for mapping URIs to verified package contents. |
| **Latest** | Highest version validly published under an identity. |

---

# 3. URI Scheme Name

```
libreseed
```

This document defines the syntax and semantics of URIs beginning with:

```
libreseed://
```

---

# 4. URI Syntax

## 4.1. General Structure

```
libreseed://<identity>/<name>[ /<version> ][ /<root-hash> ][ ?<query> ][ #<fragment> ]
```

Where:

- `<identity>` — **REQUIRED**  
- `<name>` — **REQUIRED**  
- `<version>` — OPTIONAL  
- `<root-hash>` — OPTIONAL  
- `<query>` — OPTIONAL advisory metadata  
- `<fragment>` — OPTIONAL, consumer-specific (not resolution-relevant)

---

## 4.2. Identity

`<identity>` is a multibase/multihash string derived from the author’s public key.

Example:

```
zb2rhJw92kP3nm1V...
```

Properties:

- Defines a **self-authenticating namespace**.
- All package versions published by an identity must be signed by the corresponding private key.
- No external key distribution is required.
- Prevents namespace takeover.

---

## 4.3. Package Name

`<name>` is the human-friendly name inside the identity’s namespace.

Recommended charset: `[A-Za-z0-9._-]`

Characters outside this set MUST be percent-encoded.

---

## 4.4. Version (Optional)

`<version>` is the package version string.

Examples:

```
1.2.3
1.2.3-beta.1
```

If omitted, the resolver chooses the **latest** valid version for the package under the identity.

---

## 4.5. Root Hash (Optional)

`<root-hash>` is a multibase/multihash representing the package’s content hash.

Use cases:

- Reproducible builds  
- Integrity pinning  
- Exact artifact identification

---

## 4.6. Query Component (Optional)

The query component is **advisory and unconstrained** by this specification.

Implementations MAY define their own parameters.
Resolvers MUST NOT treat query parameters as affecting package identity or resolution semantics.

---

## 4.7. Fragment Component (Optional)

The fragment is opaque to the resolver.

Example use: referencing a path inside the package.

---

# 5. Resolution Semantics

Given a `libreseed://<identity>/<name>[... ]` URI:

### 1. Identity is authoritative  
Only manifests whose public key hashes to `<identity>` are accepted.

### 2. Version handling  
- If present, resolve exactly that version.  
- If absent, resolver determines “latest.”

### 3. Hash handling  
- If `<root-hash>` is present, fetch content matching that hash.  
- If absent, discover hash via decentralized metadata.

### 4. Query and fragment  
Ignored for identity and resolution.

---

# 6. “Latest” Version Semantics

When a version is omitted:

```
libreseed://<identity>/<name>
```

The resolver:

1. Retrieves all versions for the package under `<identity>`.
2. Fetches only manifests.
3. Validates signatures (identity-bound).
4. Sorts valid versions.
5. Chooses the highest version.

This mechanism prevents version spoofing and requires no external trust bootstrap.

---

# 7. Package Manifest Requirements

Each package contains a manifest such as:

```json
{
  "identity": "zb2rhId123...",
  "name": "mypkg",
  "version": "1.2.3",
  "root_hash": "zb2rhPkgHash...",
  "public_key": "<public-key>",
  "signature": "<signature>",
  "dependencies": {
    "dep1": "1.0.0",
    "dep2": "2.3.4"
  }
}
```

Resolvers MUST verify:

1. `identity == multihash(public_key)`
2. `signature` is valid over required fields
3. `root_hash` matches actual content hash
4. Manifest is internally consistent with URI components

---

# 8. Security Considerations

- Self-authenticating namespaces prevent package spoofing.
- Hash pinning allows reproducibility.
- Version spoofing is avoided by cryptographic verification.
- No external key distribution needed.
- Query parameters MUST NOT affect identity resolution.

---

# 9. Examples

Latest version:

```
libreseed://zb2rhId789/mylib
```

Specific version:

```
libreseed://zb2rhId789/mylib/2.1.0
```

Pinned version and hash:

```
libreseed://zb2rhId789/mylib/2.1.0/zb2rhPkgHashABC...
```

Exact artifact:

```
libreseed://zb2rhId789/mylib/zb2rhPkgHashABC...
```

---

# End of Specification LS-URI-001
