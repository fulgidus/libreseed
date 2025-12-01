# LIBRESEED Protocol Specification v1.3 — NPM Bridge (Optional)

**Version:** 1.3  
**Part of:** LIBRESEED Protocol Specification

---

**Navigation:**
[← Error Handling](./LIBRESEED-SPEC-v1.3-ERROR-HANDLING.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Implementation Guide →](./LIBRESEED-SPEC-v1.3-IMPLEMENTATION-GUIDE.md)

---

## 12. NPM Bridge (Optional)

### 12.1 Overview

**LibreSeed NPM Bridge is an optional gateway layer** that allows npm clients to fetch packages from LibreSeed.

**Architecture:**
```
npm client → NPM Registry API (bridge) → LibreSeed DHT → Seeders
```

**Key Points:**
- Bridge is **NOT** part of core protocol
- Bridge is a convenience layer for npm ecosystem integration
- Bridge does not introduce centralization (stateless gateway)

---

### 12.2 Installation via NPM Bridge

```bash
# Configure npm to use bridge
npm config set registry https://libreseed-bridge.example.com

# Install package (bridge resolves via Name Index)
npm install mypackage
```

---

**Navigation:**
[← Error Handling](./LIBRESEED-SPEC-v1.3-ERROR-HANDLING.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Implementation Guide →](./LIBRESEED-SPEC-v1.3-IMPLEMENTATION-GUIDE.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
