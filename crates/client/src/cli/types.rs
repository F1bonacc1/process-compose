//! Keep in sync with `flags.go``. Only commands to start process compose and get its handle to manage.

use std::path::PathBuf;

const (
	EnvVarNamePort             = "PC_PORT_NUM"
	EnvVarNameTui              = "PC_DISABLE_TUI"
	EnvVarNameConfig           = "PC_CONFIG_FILES"
	EnvVarNameShortcuts        = "PC_SHORTCUTS_FILES"
	EnvVarNameNoServer         = "PC_NO_SERVER"
	EnvVarUnixSocketPath       = "PC_SOCKET_PATH"
	EnvVarReadOnlyMode         = "PC_READ_ONLY"
	EnvVarDisableDotEnv        = "PC_DISABLE_DOTENV"
	EnvVarTuiFullScreen        = "PC_TUI_FULL_SCREEN"
	EnvVarHideDisabled         = "PC_HIDE_DISABLED_PROC"
	EnvVarNameOrderedShutdown  = "PC_ORDERED_SHUTDOWN"
	EnvVarWithRecursiveMetrics = "PC_RECURSIVE_METRICS"
)

/// Process Compose startup flags
#[derive(Debug, Default, Clone)]
pub struct ProcessComposeFlags {
    
    /// Config file path (-f, --config)
    pub config: Option<PathBuf>,
    
    /// Port number (-p, --port)
    pub port: Option<u16>,
    
    /// Unix socket path (-u, --unix-socket)
    #[cfg(unix)]
    pub unix_socket: Option<PathBuf>,
    
    /// Use unix domain sockets (-U, --use-uds)
    #[cfg(unix)]
    pub use_uds: bool,
    
    /// Disable HTTP server (--no-server)
    pub no_server: bool,
    
    /// Enable TUI (-t, --tui)
    pub tui: Option<bool>,
    
    /// Enable TUI full screen (--tui-fs)
    pub tui_full_screen: bool,
    
    /// Environment files (-e, --env)
    pub env_files: Vec<PathBuf>,
    
    /// Namespaces (-n, --namespace)
    pub namespaces: Vec<String>,
    
    /// Hide disabled processes (-d, --hide-disabled)
    pub hide_disabled: bool,
    
    /// Refresh rate (-r, --ref-rate)
    pub refresh_rate: Option<String>,
    
    /// Slower refresh rate (--slow-ref-rate)
    pub slow_refresh_rate: Option<String>,
    
    /// Keep project running (--keep-project)
    pub keep_project: bool,
    
    /// Shortcut files (--shortcuts)
    pub shortcuts: Vec<PathBuf>,
    
    /// Disable dotenv (--disable-dotenv)
    pub disable_dotenv: bool,
    
    /// Truncate logs on startup (--logs-truncate)
    pub logs_truncate: bool,
    
    /// Recursive metrics (--recursive-metrics)
    pub recursive_metrics: bool,
    
    /// Dry run (--dry-run)
    pub dry_run: bool,
    
    /// Sort in reverse (-R, --reverse)
    pub reverse: bool,
    
    /// Sort column (-S, --sort)
    pub sort: Option<String>,
    
    /// Theme (--theme)
    pub theme: Option<String>,
    
    /// Log file path (-L, --log-file)
    pub log_file: Option<PathBuf>,
    
    /// Read only mode (--read-only)
    pub read_only: bool,
    
    /// Ordered shutdown (--ordered-shutdown)
    pub ordered_shutdown: bool,
    
    /// Detached mode (-D, --detached)
    #[cfg(unix)]
    pub detached: bool,
    
    /// Detached with TUI (--detached-with-tui)
    #[cfg(unix)]
    pub detached_with_tui: bool,
    
    /// Detach on success (--detach-on-success)
    #[cfg(unix)]
    pub detach_on_success: bool,
}

impl ProcessComposeFlags {
    /// Convert flags to command line arguments
    pub fn to_args(&self) -> Vec<String> {
        let mut args = Vec::new();
        
        if let Some(config) = &self.config {
            args.push("-f".to_string());
            args.push(config.display().to_string());
        }
        
        if let Some(port) = self.port {
            args.push("-p".to_string());
            args.push(port.to_string());
        }
        
        #[cfg(unix)]
        {
            if let Some(socket) = &self.unix_socket {
                args.push("-u".to_string());
                args.push(socket.display().to_string());
            }
            
            if self.use_uds {
                args.push("-U".to_string());
            }
            
            if self.detached {
                args.push("-D".to_string());
            }
            
            if self.detached_with_tui {
                args.push("--detached-with-tui".to_string());
            }
            
            if self.detach_on_success {
                args.push("--detach-on-success".to_string());
            }
        }
        
        if self.no_server {
            args.push("--no-server".to_string());
        }
        
        if let Some(tui) = self.tui {
            args.push(format!("-t={}", tui));
        }
        
        if self.tui_full_screen {
            args.push("--tui-fs".to_string());
        }
        
        for env_file in &self.env_files {
            args.push("-e".to_string());
            args.push(env_file.display().to_string());
        }
        
        for namespace in &self.namespaces {
            args.push("-n".to_string());
            args.push(namespace.clone());
        }
        
        if self.hide_disabled {
            args.push("-d".to_string());
        }
        
        if let Some(rate) = &self.refresh_rate {
            args.push("-r".to_string());
            args.push(rate.clone());
        }
        
        if let Some(rate) = &self.slow_refresh_rate {
            args.push("--slow-ref-rate".to_string());
            args.push(rate.clone());
        }
        
        if self.keep_project {
            args.push("--keep-project".to_string());
        }
        
        for shortcut in &self.shortcuts {
            args.push("--shortcuts".to_string());
            args.push(shortcut.display().to_string());
        }
        
        if self.disable_dotenv {
            args.push("--disable-dotenv".to_string());
        }
        
        if self.logs_truncate {
            args.push("--logs-truncate".to_string());
        }
        
        if self.recursive_metrics {
            args.push("--recursive-metrics".to_string());
        }
        
        if self.dry_run {
            args.push("--dry-run".to_string());
        }
        
        if self.reverse {
            args.push("-R".to_string());
        }
        
        if let Some(sort) = &self.sort {
            args.push("-S".to_string());
            args.push(sort.clone());
        }
        
        if let Some(theme) = &self.theme {
            args.push("--theme".to_string());
            args.push(theme.clone());
        }
        
        if let Some(log_file) = &self.log_file {
            args.push("-L".to_string());
            args.push(log_file.display().to_string());
        }
        
        if self.read_only {
            args.push("--read-only".to_string());
        }
        
        if self.ordered_shutdown {
            args.push("--ordered-shutdown".to_string());
        }
        
        args
    }
}