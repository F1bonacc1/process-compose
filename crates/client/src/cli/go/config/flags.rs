//! Keep in sync with `flags.go`. 
use std::path::PathBuf;
use std::time::Duration;


/// Default refresh interval
pub const DEFAULT_REFRESH_RATE: Duration = Duration::from_secs(1);

/// Default log level
pub const DEFAULT_LOG_LEVEL: &str = "info";

/// Default port number
pub const DEFAULT_PORT_NUM: u16 = 8080;

/// Default bind address (host)
pub const DEFAULT_ADDRESS: &str = "localhost";

/// Default log length (number of lines kept in memory)
pub const DEFAULT_LOG_LENGTH: usize = 1000;

/// Default sort column name
pub const DEFAULT_SORT_COLUMN: &str = "NAME";

/// Default theme name
pub const DEFAULT_THEME_NAME: &str = "Default";

/// Represents absence of a namespace selection
pub const NO_NAMESPACE: &str = "";


mod env {
    pub const PORT_NUM: &str = "PC_PORT_NUM";
    pub const DISABLE_TUI: &str = "PC_DISABLE_TUI";
    pub const CONFIG_FILES: &str = "PC_CONFIG_FILES";
    pub const SHORTCUTS_FILES: &str = "PC_SHORTCUTS_FILES";
    pub const NO_SERVER: &str = "PC_NO_SERVER";
    pub const SOCKET_PATH: &str = "PC_SOCKET_PATH";
    pub const READ_ONLY: &str = "PC_READ_ONLY";
    pub const DISABLE_DOTENV: &str = "PC_DISABLE_DOTENV";
    pub const TUI_FULL_SCREEN: &str = "PC_TUI_FULL_SCREEN";
    pub const HIDE_DISABLED_PROC: &str = "PC_HIDE_DISABLED_PROC";
    pub const ORDERED_SHUTDOWN: &str = "PC_ORDERED_SHUTDOWN";
    pub const RECURSIVE_METRICS: &str = "PC_RECURSIVE_METRICS";
}

/// Process Compose startup flags
#[derive(Debug, Default, Clone)]
pub struct ProcessComposeFlags {
    // Keep order in sync with Go's `Flags` in `src/config/Flags.go`

    /// Refresh rate (-r, --ref-rate)
    pub refresh_rate: Option<String>,

    /// Slower refresh rate (--slow-ref-rate)
    pub slow_refresh_rate: Option<String>,

    /// Port number (-p, --port)
    pub port: Option<u16>,

    /// Address of the server (-a, --address)
    pub address: Option<String>,

    /// Log level
    pub log_level: Option<String>,

    /// Log file path (-L, --log-file)
    pub log_file: Option<PathBuf>,

    /// Log length (lines kept in memory)
    pub log_length: Option<usize>,

    /// Follow log output (-f, --follow)
    pub log_follow: bool,

    /// Tail length (-n, --tail)
    pub log_tail_length: Option<usize>,

    /// Output raw logs (--raw-log)
    pub raw_log: bool,

    /// Enable TUI (-t, --tui)
    pub tui: Option<bool>,

    /// Command to run
    pub command: Option<String>,

    /// Write changes
    pub write: bool,

    /// Ignore dependencies
    pub no_dependencies: bool,

    /// Hide disabled processes (-d, --hide-disabled)
    pub hide_disabled: bool,

    /// Sort column (-S, --sort)
    pub sort: Option<String>,

    /// Sort column changed
    pub sort_changed: bool,

    /// Sort in reverse (-R, --reverse)
    pub reverse: bool,

    /// Disable HTTP server (--no-server)
    pub no_server: bool,

    /// Keep TUI on
    pub keep_tui_on: bool,

    /// Keep project running (--keep-project)
    pub keep_project: bool,

    /// Ordered shutdown (--ordered-shutdown)
    pub ordered_shutdown: bool,

    /// Theme (--theme)
    pub theme: Option<String>,

    /// Theme changed
    pub theme_changed: bool,

    /// Shortcut files (--shortcuts)
    pub shortcuts: Vec<PathBuf>,

    /// Unix socket path (-u, --unix-socket)
    #[cfg(unix)]
    pub unix_socket: Option<PathBuf>,

    /// Use unix domain sockets (-U, --use-uds)
    #[cfg(unix)]
    pub use_uds: bool,

    /// Read only mode (--read-only)
    pub read_only: bool,

    /// Output format
    pub output_format: Option<String>,

    /// Disable dotenv (--disable-dotenv)
    pub disable_dotenv: bool,

    /// Enable TUI full screen (--tui-fs)
    pub tui_full_screen: bool,

    /// Detached mode (-D, --detached)
    #[cfg(unix)]
    pub detached: bool,

    /// Detached with TUI (--detached-with-tui)
    #[cfg(unix)]
    pub detached_with_tui: bool,

    /// Namespaces (-n, --namespace)
    pub namespaces: Vec<String>,

    /// Detach on success (--detach-on-success)
    #[cfg(unix)]
    pub detach_on_success: bool,

    /// Wait for project to be ready (--wait)
    pub wait_ready: bool,

    /// Print short version (--short)
    pub short_version: bool,

    /// Truncate logs on startup (--logs-truncate)
    pub logs_truncate: bool,

    /// Recursive metrics (--recursive-metrics)
    pub recursive_metrics: bool,

    // Rust-specific extras (not present in Go's Flags)

    /// Config file path (-f, --config)
    pub config: Option<PathBuf>,

    /// Environment files (-e, --env)
    pub env_files: Vec<PathBuf>,

    /// Dry run (--dry-run)
    pub dry_run: bool,
}
