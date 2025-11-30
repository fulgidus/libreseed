# Feature Specification: CLI Rename and Installation Script

**Feature Branch**: `002-cli-rename-install`  
**Created**: 2025-11-30  
**Status**: Draft  
**Input**: User description: "rename libreseed-daemon to lbsd (LiBreSeedDaemon), rename cli command to just lbs (e.g., lbs start), and add a bash install.sh for easy installation and updates"

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

- What happens when the user runs the install script without sufficient disk space?
- How does the system handle installation on unsupported architectures (ARM, MacOS)?
- What happens if the compilation fails during installation?
- How does the CLI handle being invoked with invalid subcommands?
- What happens if the daemon crashes immediately after `lbs start`?
- How does the update process handle breaking configuration changes?
- What happens if `/usr/local/bin` is not in the user's PATH?
- How does the system handle multiple simultaneous installations (race conditions)?

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

**CLI Requirements**:

- **FR-008**: The CLI command MUST be renamed from current name to `lbs`
- **FR-009**: The `lbs` command MUST support subcommand `start` to start the daemon
- **FR-010**: The `lbs` command MUST support subcommand `stop` to stop the daemon
- **FR-011**: The `lbs` command MUST support subcommand `status` to show daemon status
- **FR-012**: The `lbs` command MUST support subcommand `restart` to restart the daemon
- **FR-013**: The `lbs` command MUST display help/usage information when invoked without arguments
- **FR-014**: The `lbs` command MUST support `--version` flag to display version information

**Daemon Requirements**:

- **FR-015**: The daemon binary MUST be renamed from `libreseed-daemon` to `lbsd`
- **FR-016**: The daemon MUST maintain all existing functionality after rename
- **FR-017**: The daemon MUST use the same configuration file locations as before

**Update Requirements**:

- **FR-018**: Running `install.sh` on an existing installation MUST update binaries to the latest version
- **FR-019**: Updates MUST preserve existing configuration files
- **FR-020**: The update process MUST warn if the daemon is running and offer to stop it

**Uninstall Requirements**:

- **FR-021**: The `install.sh` script MUST support `--uninstall` flag to remove the installation
- **FR-022**: Uninstallation MUST remove `lbs` and `lbsd` binaries from `/usr/local/bin`
- **FR-023**: Uninstallation MUST stop the daemon if it is running
- **FR-024**: Uninstallation MUST ask user whether to preserve or remove configuration files

### Key Entities *(include if feature involves data)*

- **lbs (CLI Binary)**: Command-line interface binary, provides user-facing commands (start, stop, status, restart), communicates with daemon
- **lbsd (Daemon Binary)**: Background service binary, runs as system daemon, manages LibreSeed core functionality
- **install.sh (Installation Script)**: Bash script for installation, update, and uninstallation workflows
- **Configuration Files**: User and system configuration (preserved across updates, optionally preserved on uninstall)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can install LibreSeed from source using `./install.sh` in under 5 minutes on a system with Go installed
- **SC-002**: After installation, `lbs start`, `lbs stop`, `lbs status`, and `lbs restart` commands work correctly 100% of the time
- **SC-003**: The installation script succeeds on at least Ubuntu 20.04+, Debian 11+, and Fedora 35+
- **SC-004**: 95% of installations complete without errors on supported platforms
- **SC-005**: Updates preserve existing configuration files with 100% reliability
- **SC-006**: The CLI responds to user commands within 3 seconds under normal conditions
- **SC-007**: Users can uninstall LibreSeed completely, leaving no binaries in `/usr/local/bin`
- **SC-008**: Installation script provides actionable error messages for all common failure scenarios (missing Go, insufficient permissions, etc.)
- **SC-009**: The shortened command names (`lbs`, `lbsd`) reduce command-typing time by at least 40% compared to previous names
- **SC-010**: Zero breaking changes to daemon functionality or configuration format

## Assumptions *(if any)*

- Users have Go 1.21+ installed on their system (or are willing to install it)
- Users have sudo/root access or `/usr/local/bin` is writable
- The project structure remains compatible with standard `go build` commands
- The target platform is Linux (primary support; BSD/MacOS may work but not guaranteed)
- Users are comfortable running bash scripts for installation
- Network connectivity is available for downloading dependencies during `go build`

## Out of Scope *(if any)*

- Packaging for distribution-specific package managers (apt, yum, pacman) - future enhancement
- Systemd service file installation and management - future enhancement
- Windows support for installation script (Windows users must build manually)
- Automatic update checking or notification - future enhancement
- Installation to custom directories (always uses `/usr/local/bin`)
- Daemon configuration migration for breaking changes - future enhancement
- Binary signing or verification - future enhancement
- Rollback functionality if update fails - future enhancement

## Open Questions *(if any)*

*None - all key decisions have been made with reasonable defaults.*

## Dependencies

- Go 1.21+ must be installed on the target system
- Bash shell must be available for running `install.sh`
- Standard Unix utilities: `chmod`, `which`, `mkdir`, `rm`, `cp`
- Sufficient disk space for compilation and binary installation (~50MB)

## Technical Notes

- The CLI binary (`lbs`) should be the main entry point; daemon binary (`lbsd`) may be invoked by CLI or run standalone
- Consider whether `lbs start` should fork `lbsd` or if `lbsd` should be run directly by users (decision needed during implementation)
- Installation script should be idempotent (running multiple times produces same result)
- Version information should be embedded in binaries at compile time using `-ldflags`
- Configuration file compatibility should be maintained (no breaking changes to format)

## Related Features

*None identified at this time.*

## References

- Original request: Rename daemon to `lbsd`, CLI to `lbs`, add `install.sh`
- Project: LibreSeed - Decentralized package distribution system
- Current state: Existing codebase with `libreseed-daemon` and CLI implementation
