# Feature Specification: CLI Rename and Installation Script

**Feature Branch**: `002-cli-rename-install`  
**Created**: 2025-11-30  
**Status**: Draft  
**Input**: User description: "rename libreseed-daemon to lbsd (LiBreSeedDaemon), rename cli command to just lbs (e.g., lbs start), and add a bash install.sh for easy installation and updates"

## Clarifications

### Session 2025-11-30

- Q: How should `lbs start` invoke the daemon? → A: Option B - `lbs start` forks `lbsd` as a background process and monitors its PID
- Q: How should `install.sh` handle failures during installation? → A: Option B - Transactional Install (Full Rollback) - Track all changes and roll back completely on any failure
- Q: How should the system behave when the daemon crashes after successful startup? → A: Option C - Supervisor Integration (Future-Ready) - CLI handles only manual start/stop, users should use systemd/supervisor for production auto-restart needs
- Q: What logging and diagnostics approach should `install.sh` and `lbs` provide? → A: Option A - Minimal (STDOUT/STDERR Only) with future OpenTelemetry integration for remote logging and performance diagnostics
- Q: How should users verify binary integrity and authenticity? → A: Option E with Option B as future target - For v0.2.0: SHA256 checksums only (downloaded from GitHub release). Future enhancement: GPG-signed SHA256SUMS file for cryptographic verification. The installer will verify checksums to detect corruption/incomplete downloads but will not verify signatures in this version

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Easy Installation (Priority: P1)

As a LibreSeed user, I want to install the daemon and CLI tools using a single bash script, so that I can quickly get started without complex manual setup steps.

**Why this priority**: This is the foundation for all other functionality. Without a working installation mechanism, users cannot use any features. A smooth installation experience is critical for user adoption.

**Independent Test**: Can be fully tested by running `./install.sh` on a clean system and verifying that `lbs` and `lbsd` commands are available and functional.

**Acceptance Scenarios**:

1. **Given** a Linux system with Go installed, **When** I run `./install.sh`, **Then** the script compiles the project, installs `lbsd` and `lbs` binaries to `/usr/local/bin`, and displays success messages
2. **Given** a system without Go installed, **When** I run `./install.sh`, **Then** the script detects the missing dependency and provides clear instructions on how to install Go
3. **Given** a system where `/usr/local/bin` requires sudo, **When** I run `./install.sh`, **Then** the script prompts for sudo password only when needed and explains why
4. **Given** an existing installation, **When** I run `./install.sh`, **Then** the script detects the existing installation and asks for confirmation before overwriting

---

### User Story 2 - Daemon Management with New CLI (Priority: P1)

As a LibreSeed user, I want to control the daemon using short, memorable commands like `lbs start` and `lbs stop`, so that I can efficiently manage the service without typing long command names.

**Why this priority**: This is the core user interaction after installation. The CLI rename directly addresses the user's request for improved usability and must work immediately after installation.

**Independent Test**: Can be fully tested by installing the system and running `lbs start`, `lbs stop`, `lbs status`, and `lbs restart` commands, verifying each produces the expected daemon state changes.

**Acceptance Scenarios**:

1. **Given** the daemon is not running, **When** I run `lbs start`, **Then** the daemon starts successfully and confirms startup
2. **Given** the daemon is running, **When** I run `lbs stop`, **Then** the daemon stops gracefully and confirms shutdown
3. **Given** the daemon is in any state, **When** I run `lbs status`, **Then** I see the current daemon status (running/stopped) and key metrics
4. **Given** the daemon is running, **When** I run `lbs restart`, **Then** the daemon stops gracefully and starts again
5. **Given** I type `lbs` with no arguments, **When** the command executes, **Then** I see usage help with available subcommands
6. **Given** the daemon crashes after `lbs start`, **When** I run `lbs status`, **Then** I see that the daemon is not running and the last known status

---

### User Story 3 - Easy Updates (Priority: P2)

As a LibreSeed user, I want to update the daemon and CLI to the latest version using the same installation script, so that I can keep my system current without manual binary management.

**Why this priority**: Regular updates are important for security and features, but less critical than initial installation and basic operation. Users need this after they've been using the system for a while.

**Independent Test**: Can be fully tested by installing an older version, modifying VERSION file, running `./install.sh`, and verifying the new version is installed while preserving configuration.

**Acceptance Scenarios**:

1. **Given** an existing LibreSeed installation, **When** I run `./install.sh` with a newer version, **Then** the script updates the binaries and preserves my existing configuration
2. **Given** the daemon is running during update, **When** I run `./install.sh`, **Then** the script warns me and asks if I want to stop the daemon before updating
3. **Given** a successful update, **When** I run `lbs --version`, **Then** I see the new version number

---

### User Story 4 - Clean Uninstallation (Priority: P3)

As a LibreSeed user, I want to cleanly uninstall the daemon and CLI tools, so that I can remove the software completely if I no longer need it.

**Why this priority**: Uninstallation is important for completeness and user trust, but users need installation and operation first. This is a lower-frequency operation.

**Independent Test**: Can be fully tested by installing LibreSeed, running `./install.sh --uninstall`, and verifying all binaries and system artifacts are removed.

**Acceptance Scenarios**:

1. **Given** an installed LibreSeed system, **When** I run `./install.sh --uninstall`, **Then** the script removes `lbs` and `lbsd` binaries and asks if I want to remove configuration files
2. **Given** the daemon is running during uninstall, **When** I run `./install.sh --uninstall`, **Then** the script stops the daemon before removing files
3. **Given** configuration preservation is requested, **When** uninstall completes, **Then** config files remain in place for future reinstallation

---

### Edge Cases

- What happens when the user runs the install script with less than 50MB available disk space in /usr/local/bin? → Transactional install will detect insufficient space and roll back
- How does the system handle installation on unsupported architectures (ARM, MacOS)? → Out of scope, best-effort
- What happens if the compilation fails during installation? → Transactional install will roll back automatically
- How does the CLI handle being invoked with invalid subcommands? → Display usage help
- What happens if the daemon crashes immediately after `lbs start`? → Detected via PID monitoring in SC-011
- How does the update process handle breaking configuration changes? → Out of scope for this feature
- What happens if `/usr/local/bin` is not in the user's PATH? → Installation succeeds but user must add to PATH manually
- How does the system handle multiple simultaneous installations (race conditions)? → Not addressed (low-priority edge case)
- What happens if network connectivity fails during `go build` or checksum download? → Transactional install detects failure and rolls back; user should check network and retry

## Requirements *(mandatory)*

### Functional Requirements

**Installation Requirements**:

- **FR-001**: The `install.sh` script MUST check for Go installation and provide clear error messages if not found
- **FR-002**: The `install.sh` script MUST compile the project using `go build` for both daemon and CLI
- **FR-003**: The `install.sh` script MUST install `lbsd` binary (daemon) to `/usr/local/bin/lbsd`
- **FR-004**: The `install.sh` script MUST install `lbs` binary (CLI) to `/usr/local/bin/lbs`
- **FR-005**: The `install.sh` script MUST detect existing installations and prompt for confirmation before overwriting
- **FR-006**: The `install.sh` script MUST set appropriate executable permissions (chmod +x) on installed binaries
- **FR-007**: The `install.sh` script MUST provide clear success/failure messages at each step
- **FR-026**: The `install.sh` script MUST implement transactional installation: track all modifications (binaries installed, configs modified, directories created) and automatically roll back ALL changes if any step fails
- **FR-027**: On installation failure, the script MUST restore the system to its exact pre-install state (remove binaries, restore original shell configs, clean temporary files)
- **FR-028**: The rollback mechanism MUST execute automatically without user intervention and log all rollback actions
- **FR-031**: The `install.sh` script MUST output all messages to STDOUT (success/info) and STDERR (errors/warnings) only, without creating log files
- **FR-033**: The `install.sh` script MUST download `SHA256SUMS` file from GitHub release for the target version
- **FR-034**: The `install.sh` script MUST verify downloaded binaries against checksums before installation
- **FR-035**: Installation MUST fail if checksum verification fails, with clear error message indicating which file failed verification
- **FR-036**: Checksum verification errors MUST provide actionable guidance (re-download, check network connection, report issue)

**CLI Requirements**:

- **FR-008**: The CLI command MUST be renamed from current name to `lbs`
- **FR-009**: The `lbs start` command MUST fork `lbsd` as a background process, monitor its PID, and verify successful startup
- **FR-010**: The `lbs` command MUST support subcommand `stop` to stop the daemon by sending termination signal to the monitored PID
- **FR-011**: The `lbs` command MUST support subcommand `status` to show daemon status by checking if the monitored PID is alive
- **FR-012**: The `lbs` command MUST support subcommand `restart` to stop and then start the daemon
- **FR-013**: The `lbs` command MUST display help/usage information when invoked without arguments
- **FR-014**: The `lbs` command MUST support `--version` flag to display version information
- **FR-015**: The `lbs` command MUST persist the daemon PID to a file (e.g., `/var/run/lbsd.pid` or `~/.local/share/libreseed/lbsd.pid`) for status tracking across CLI invocations
- **FR-029**: The CLI MUST NOT implement automatic restart on daemon crash (manual restart only via `lbs start`)
- **FR-030**: The `lbs status` output MUST clearly indicate if the daemon has crashed (PID file exists but process not running)
- **FR-032**: The `lbs` command MUST output all messages to STDOUT (success/info) and STDERR (errors/warnings) only, without creating log files

**Daemon Requirements**:

- **FR-016**: The daemon binary MUST be renamed from `libreseed-daemon` to `lbsd`
- **FR-017**: The daemon MUST maintain all existing functionality after rename
- **FR-018**: The daemon MUST use the same configuration file locations as before

**Update Requirements**:

- **FR-019**: Running `install.sh` on an existing installation MUST update binaries to the latest version
- **FR-020**: Updates MUST preserve existing configuration files
- **FR-021**: The update process MUST warn if the daemon is running and offer to stop it

**Uninstall Requirements**:

- **FR-022**: The `install.sh` script MUST support `--uninstall` flag to remove the installation
- **FR-023**: Uninstallation MUST remove `lbs` and `lbsd` binaries from `/usr/local/bin`
- **FR-024**: Uninstallation MUST stop the daemon if it is running
- **FR-025**: Uninstallation MUST ask user whether to preserve or remove configuration files

**Error Handling Requirements**:

- **FR-037**: Both `install.sh` and `lbs` MUST use standardized exit codes: 0 (success), 1 (user error - invalid input, missing dependencies), 2 (system error - permissions, disk space, process failures), 3 (configuration error - invalid config files)
- **FR-038**: All error messages MUST follow the format: "[ERROR] <component>: <problem description> - <actionable guidance>"
- **FR-039**: Error messages MUST provide actionable next steps (e.g., "Install Go 1.21+ using your package manager: apt install golang-go" or "Run with sudo: sudo ./install.sh")
- **FR-040**: The `lbs` command MUST validate all user inputs (subcommands, flags) and provide specific error messages for invalid usage rather than generic "command not found" messages

### Key Entities *(include if feature involves data)*

- **lbs (CLI Binary)**: Command-line interface binary, provides user-facing commands (start, stop, status, restart), manages daemon lifecycle by forking `lbsd` and tracking its PID
- **lbsd (Daemon Binary)**: Background service binary, runs as system daemon, manages LibreSeed core functionality
- **PID File**: File storing the daemon process ID for lifecycle management (e.g., `/var/run/lbsd.pid` or `~/.local/share/libreseed/lbsd.pid`)
- **install.sh (Installation Script)**: Bash script for installation, update, and uninstallation workflows
- **Configuration Files**: User and system configuration (preserved across updates, optionally preserved on uninstall)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can install LibreSeed from source using `./install.sh` with `go build` completing in under 2 minutes and full installation completing in under 5 minutes on a system with Go installed
- **SC-002**: After installation, `lbs start`, `lbs stop`, `lbs status`, and `lbs restart` commands work correctly 100% of the time
- **SC-003**: The installation script succeeds on at least Ubuntu 20.04+, Debian 11+, and Fedora 35+
- **SC-004**: 95% of installations complete without errors on supported platforms
- **SC-005**: Updates preserve existing configuration files with 100% reliability
- **SC-006**: The CLI responds to user commands within 3 seconds under normal conditions
- **SC-007**: Users can uninstall LibreSeed completely, leaving no binaries in `/usr/local/bin`
- **SC-008**: Installation script provides actionable error messages for all common failure scenarios (missing Go, insufficient permissions, etc.)
- **SC-009**: The shortened command names (`lbs`, `lbsd`) reduce command-typing time by at least 40% compared to previous names
- **SC-010**: Zero breaking changes to daemon functionality or configuration format
- **SC-011**: `lbs start` successfully detects and reports daemon startup failures within 2 seconds
- **SC-012**: Installation failures trigger automatic rollback with 100% success rate, leaving no partial artifacts on the system

## Assumptions *(if any)*

- Users have Go 1.21+ installed on their system (or are willing to install it)
- Users have sudo/root access or `/usr/local/bin` is writable
- The project structure remains compatible with standard `go build` commands
- The target platform is Linux (primary support; BSD/MacOS may work but not guaranteed)
- Users are comfortable running bash scripts for installation
- Network connectivity is available for downloading dependencies during `go build`
- Users requiring automatic daemon restart will use systemd, supervisor, or similar tools (not provided by this feature)

## Testing Strategy *(mandatory)*

This feature uses **manual validation** as the primary testing approach, which satisfies the constitution's "test with rigor" principle through comprehensive acceptance scenarios and edge case validation.

### Testing Approach Rationale

- **Installation Testing**: Installation and system configuration cannot be effectively unit-tested. Manual validation on real systems is the industry standard for installation scripts.
- **Process Lifecycle Testing**: Daemon forking, PID management, and process monitoring require real system processes and cannot be reliably mocked.
- **Platform Validation**: Cross-platform behavior (Ubuntu, Debian, Fedora) must be validated on actual systems.

### Testing Coverage

1. **Acceptance Scenarios**: Each user story includes complete acceptance scenarios with Given/When/Then structure (US1-US4)
2. **Edge Cases**: 9 documented edge cases covering disk space, architecture support, compilation failure, invalid commands, crashes, breaking changes, PATH issues, race conditions, and network failures
3. **Success Criteria**: 12 measurable success criteria (SC-001 through SC-012) with quantified thresholds
4. **Manual Validation Tasks**: Tasks T051-T053 in tasks.md validate performance, startup detection, and rollback criteria

### Validation Checkpoints

- **Phase 3 Checkpoint**: Validate US1 (Installation) independently on clean systems (Ubuntu 20.04+, Debian 11+, Fedora 35+)
- **Phase 4 Checkpoint**: Validate US2 (CLI) independently by testing all subcommands (start, stop, status, restart, stats)
- **Phase 5 Checkpoint**: Validate US3 (Updates) by installing older version, updating, verifying config preservation
- **Phase 6 Checkpoint**: Validate US4 (Uninstall) by installing, uninstalling, verifying complete cleanup
- **Phase 7 Validation**: Execute T051-T053 to verify performance and edge case criteria

This manual validation strategy is **rigorous, comprehensive, and appropriate for system-level installation and process management features**.

---

## Out of Scope *(if any)*

- Packaging for distribution-specific package managers (apt, yum, pacman) - future enhancement
- Systemd service file installation and management - future enhancement (recommended for production use)
- Windows support for installation script (Windows users must build manually)
- Automatic update checking or notification - future enhancement
- Installation to custom directories (always uses `/usr/local/bin`)
- Daemon configuration migration for breaking changes - future enhancement
- Rollback functionality if update fails - future enhancement
- Automatic daemon restart on crash - users should use systemd/supervisor for production environments requiring auto-restart
- Log files for install.sh or CLI operations - users can redirect STDOUT/STDERR to files if needed
- OpenTelemetry integration for remote logging and performance diagnostics - future enhancement
- GPG/Minisign signature verification (future enhancement)

## Open Questions *(if any)*

*None - all key decisions have been made with reasonable defaults.*

## Dependencies

- Go 1.21+ must be installed on the target system
- Bash shell must be available for running `install.sh`
- Standard Unix utilities: `chmod`, `which`, `mkdir`, `rm`, `cp`
- Sufficient disk space for compilation and binary installation (~50MB)
- Minimum 512MB RAM available for Go compilation (1GB+ recommended)

## Technical Notes

- The CLI binary (`lbs`) is the main entry point and manages daemon lifecycle
- `lbs start` forks `lbsd` as a background process and monitors its PID for lifecycle management
- The daemon PID is persisted to a file for cross-invocation status tracking
  - Recommended location: `~/.local/share/libreseed/lbsd.pid` (user-specific, no sudo required)
  - Alternative: `/var/run/lbsd.pid` (system-wide, requires sudo/root permissions)
  - Implementation should prefer user-specific path for accessibility
- Installation script should be idempotent (running multiple times produces same result)
- Version information should be embedded in binaries at compile time using `-ldflags`
- Configuration file compatibility should be maintained (no breaking changes to format)
- For production environments requiring automatic restart on crash, users should configure systemd or supervisor (example systemd unit file should be provided in documentation as future enhancement)
- All output from `install.sh` and `lbs` goes to STDOUT/STDERR; users can redirect to files using standard shell redirection (e.g., `./install.sh 2>&1 | tee install.log`)
- Future consideration: OpenTelemetry endpoint integration for remote logging, tracing, and performance diagnostics
- Checksum verification provides integrity checking (corruption/incomplete downloads) but not authenticity verification (compromised releases). Cryptographic signing (GPG) deferred to future version

## Related Features

*None identified at this time.*

## References

- Original request: Rename daemon to `lbsd`, CLI to `lbs`, add `install.sh`
- Project: LibreSeed - Decentralized package distribution system
- Current state: Existing codebase with `libreseed-daemon` and CLI implementation

---

## Process Learnings & Experience Encoding *(Session 2025-11-30)*

### Clarification Process Insights

**Pattern: Multi-Option Decision Framework**  
The clarification process benefited significantly from presenting **structured options with clear trade-offs** rather than yes/no questions. For example:
- **Process forking question**: Offering three distinct approaches (Option A: External systemd-only, Option B: CLI-managed fork, Option C: Future supervisor) enabled rapid, informed decision-making.
- **Rollback strategy question**: Presenting failure handling options (Best-Effort vs Transactional vs No-Rollback) clarified user priorities early and prevented scope creep.

**Learning**: When clarifying requirements, **enumerate concrete alternatives with pros/cons** rather than asking open-ended questions. This accelerates decision velocity and surfaces hidden assumptions.

---

**Pattern: Progressive Disclosure of Complexity**  
Questions were ordered from **high-level architecture decisions to implementation details**:
1. First: Process management approach (fundamental architecture)
2. Second: Failure handling philosophy (system behavior contract)
3. Third: Supervisor integration (operational context)
4. Finally: Logging and integrity (implementation specifics)

**Learning**: **Defer low-level implementation questions until architectural fundamentals are locked**. This prevents premature optimization and allows later decisions to be informed by earlier constraints.

---

**Pattern: Explicit Future-Proofing vs Current Scope**  
Several decisions intentionally **deferred complexity to future versions** while establishing clear extension points:
- Supervisor integration marked as "future-ready" (Option C), establishing clear boundary between CLI's manual control and production automation needs
- OpenTelemetry mentioned as "future enhancement" while keeping current logging minimal
- GPG signing deferred but SHA256 checksums implemented as stepping stone

**Learning**: **Explicitly name "future enhancement" items during spec creation** to prevent feature creep while documenting intentional technical debt. This creates upgrade path clarity without current-version bloat.

---

### Constitution Gate Application

**Pattern: Proactive Gate Evaluation**  
The constitution gate was evaluated **during planning phase (Phase 0) rather than just before implementation**. This early evaluation:
- Surface test strategy gaps (initially no test approach defined)
- Clarified acceptance scenarios needed refinement for independence
- Identified missing rollback requirements that became FR-026/FR-027/FR-028

**Learning**: **Apply constitution gate as early as clarification phase**—treat it as requirements validation, not implementation gatekeeper. Fixes specification gaps before design begins.

---

**Pattern: Test Independence as Specification Clarity Metric**  
Principle #2 ("test independence") forced clarification of **feature boundaries and dependencies**:
- User Story 1 (Installation): Initially unclear if it depended on daemon functionality
- Refined to: "Can be tested by running `./install.sh` and verifying binaries are available"—**no runtime behavior validation needed**
- This clarification separated installation concerns from daemon lifecycle concerns

**Learning**: **If acceptance scenarios aren't independently testable, the feature boundaries are unclear**. Use test independence as a specification smell detector.

---

**Pattern: Complexity Tracking as Scope Control**  
Principle #5 (complexity bounds) was applied as **active scope limiter**:
- Identified 7 files modified, ~500-1000 LOC
- Explicitly documented "no new architectural patterns"
- Called out "best-effort ARM/MacOS support" to prevent cross-platform scope explosion

**Learning**: **Track complexity metrics during specification, not just implementation**. This creates early warning system for scope creep before code is written.

---

### Collaboration & Decision Recording

**Pattern: Decision Rationale Capture**  
Each clarification question captured **not just the decision but the rationale**:
- Example: "Option B (CLI-managed fork)" chosen **because** it provides manual control without external dependencies, balancing simplicity with future systemd integration
- Example: Transactional install chosen **because** partial installation failures create support burden and user confusion

**Learning**: **Record "why" alongside "what" in clarifications section**. Future contributors need decision context, not just outcomes. This prevents requirement churn from "why did we choose this?" questions.

---

**Pattern: Explicit Out-of-Scope Documentation**  
The specification explicitly lists **what is NOT being built** and why:
- Windows support → Platform complexity exceeds value for v0.2.0
- Automatic update checking → Network complexity, user control preference
- GPG signing → Deferred to future version with clear upgrade path

**Learning**: **Out-of-scope section is as important as requirements section**. It documents negative decisions and prevents feature resurrection during implementation.

---

### Edge Case Handling Philosophy

**Pattern: Edge Case Triage During Specification**  
Edge cases were **explicitly enumerated and triaged** during spec creation:
- High-priority: Handled in requirements (disk space failure → transactional rollback)
- Medium-priority: Documented but deferred (breaking config changes → out of scope)
- Low-priority: Acknowledged but accepted (simultaneous installations → race condition not addressed)

**Learning**: **Don't silently ignore edge cases; explicitly triage them**. This prevents "what about X?" questions during implementation and creates clear acceptance boundaries.

---

### Success Criteria as Behavioral Contract

**Pattern: Quantified Expectations**  
Success criteria were defined with **concrete, measurable numbers**:
- SC-001: "Under 5 minutes" (not "quickly")
- SC-004: "95% of installations" (not "most")
- SC-006: "Within 3 seconds" (not "responsive")
- SC-009: "40% reduction in typing time" (not "shorter commands")

**Learning**: **Vague success criteria create implementation uncertainty**. Force quantification during specification—if you can't measure it, you can't verify it.

---

**Pattern: Negative Success Criteria**  
SC-010 defines success as **absence of change**: "Zero breaking changes to daemon functionality or configuration format"

**Learning**: **Success isn't always adding functionality; sometimes it's preserving stability**. Explicitly state "do no harm" requirements as success criteria.

---

### Specification Structure Lessons

**Pattern: Mandatory Section Enforcement**  
The specification enforced **mandatory sections** (User Scenarios, Requirements, Success Criteria):
- Prevented incomplete specs (can't skip acceptance scenarios)
- Created consistent review structure across features
- Forced early thinking about testability

**Learning**: **Template-driven specifications with mandatory sections prevent "figure it out during implementation" syndrome**. Empty optional sections are okay; missing mandatory sections are blockers.

---

**Pattern: Priority Labeling with Rationale**  
Each user story included **priority (P1/P2/P3) with explicit justification**:
- P1: Installation → "Without this, users cannot use any features"
- P2: Updates → "Less critical than initial installation"
- P3: Uninstallation → "Lower-frequency operation"

**Learning**: **Priority without rationale is arbitrary**. Force justification during specification to prevent priority inflation and enable intelligent scope cuts.

---

### Key Takeaways for Future Feature Specifications

1. **Front-load architectural decisions** via structured multi-option questions (don't dive into implementation details first)
2. **Apply constitution gate during specification**, not just before code—use it to validate requirements clarity
3. **Explicitly document negative decisions** (out-of-scope, deferred, rejected options)
4. **Quantify all success criteria**—vague expectations create implementation paralysis
5. **Test independence is a specification smell detector**—if scenarios aren't independently testable, refine feature boundaries
6. **Record decision rationale**, not just outcomes—future maintainers need context
7. **Triage edge cases explicitly**—don't leave them as implicit "implementation details"
8. **Use complexity tracking as scope control**, not just implementation metric
9. **Priority labels require justification**—prevent arbitrary prioritization
10. **Treat mandatory specification sections as requirements validation checklist**, not bureaucratic overhead

---

*These learnings encode process patterns, collaboration insights, and decision-making frameworks discovered during Phase 0. They represent reusable specification practices, not technical implementation details (which belong in research.md and plan.md).*
