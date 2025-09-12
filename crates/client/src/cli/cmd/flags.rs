//! Keep in sync with `flags.go`.
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

pub mod env {
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
    pub const LOG_FILE: &str = "PC_LOG_FILE";
}
