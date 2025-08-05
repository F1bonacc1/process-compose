use clap::Parser;
use std::num::NonZero;
use std::path::PathBuf;

use crate::cli::cmd::flags::env;
use crate::cli::cmd::flags::{DEFAULT_REFRESH_RATE, DEFAULT_SORT_COLUMN, DEFAULT_THEME_NAME};

#[derive(Debug, Clone, Parser, struct_field_names::StructFieldNames, bon::Builder)]
pub struct ProcessComposeFlagsUp {
    /// Path to config files to load (env: PC_CONFIG_FILES)
    #[arg(long = "config", env = env::CONFIG_FILES)]
    #[builder(default = Vec::new())]
    pub config: Vec<PathBuf>,

    /// Detach the TUI after successful startup (requires --detached-with-tui)
    #[arg(long = "detach-on-success", default_value_t = false)]
    #[builder(default = false)]
    pub detach_on_success: bool,

    /// Run in detached mode
    #[cfg(unix)]
    #[arg(long = "detached", default_value_t = false)]
    #[builder(default = false)]
    pub detached: bool,

    /// Run in detached mode with TUI
    #[cfg(unix)]
    #[arg(long = "detached-with-tui", default_value_t = false)]
    #[builder(default = false)]
    pub detached_with_tui: bool,

    /// Disable .env file loading (env: PC_DISABLE_DOTENV=1)
    #[arg(long = "disable-dotenv", env = env::DISABLE_DOTENV, default_value_t = false)]
    #[builder(default = false)]
    pub disable_dotenv: bool,

    /// Validate the config and exit
    #[arg(long = "dry-run", default_value_t = false)]
    #[builder(default = false)]
    pub dry_run: bool,

    /// Path to env files to load (default .env)
    #[arg(long = "env", default_value = ".env")]
    #[builder(default = vec![PathBuf::from(".env")])]
    pub env_files: Vec<PathBuf>,

    /// Hide disabled processes (env: PC_HIDE_DISABLED_PROC)
    #[arg(long = "hide-disabled", env = env::HIDE_DISABLED_PROC, default_value_t = false)]
    #[builder(default = false)]
    pub hide_disabled: bool,

    /// Keep the project running even after all processes exit
    #[arg(long = "keep-project", default_value_t = false)]
    #[builder(default = false)]
    pub keep_project: bool,

    /// Truncate process logs buffer on startup
    #[arg(long = "logs-truncate", default_value_t = false)]
    #[builder(default = false)]
    pub logs_truncate: bool,

    /// Run only specified namespaces (default: all)
    #[arg(long = "namespace")]
    #[builder(default = Vec::new())]
    pub namespaces: Vec<String>,

    /// Do not start dependent processes
    #[arg(long = "no-deps", default_value_t = false)]
    #[builder(default = false)]
    pub no_deps: bool,

    /// Collect metrics recursively (env: PC_RECURSIVE_METRICS)
    #[arg(long = "recursive-metrics", env = env::RECURSIVE_METRICS, default_value_t = false)]
    #[builder(default = false)]
    pub recursive_metrics: bool,

    /// TUI refresh interval in seconds
    #[arg(long = "ref-rate", default_value_t = DEFAULT_REFRESH_RATE.as_secs().try_into().unwrap())]
    #[builder(default = NonZero::new(DEFAULT_REFRESH_RATE.as_secs()).unwrap())]
    pub refresh_rate: NonZero<u64>,

    /// Sort in reverse order
    #[arg(long = "reverse", default_value_t = false)]
    #[builder(default = false)]
    pub reverse: bool,

    /// Paths to shortcut config files to load (env: PC_SHORTCUTS_FILES)
    #[arg(long = "shortcuts", env = env::SHORTCUTS_FILES)]
    #[builder(default = Vec::new())]
    pub shortcuts: Vec<PathBuf>,

    /// Slow(er) refresh interval for resource metrics (must be > --ref-rate)
    #[arg(long = "slow-ref-rate")]
    pub slow_refresh_rate: Option<String>,

    /// Sort column name (default NAME)
    #[arg(long = "sort", default_value = DEFAULT_SORT_COLUMN)]
    #[builder(default = DEFAULT_SORT_COLUMN.to_string())]
    pub sort: String,

    /// Select process compose theme (default Default)
    #[arg(long = "theme", default_value = DEFAULT_THEME_NAME)]
    #[builder(default = DEFAULT_THEME_NAME.to_string())]
    pub theme: String,

    /// Enable / disable TUI (use --tui=false to disable) (env: PC_DISABLE_TUI)
    #[arg(long = "tui", env = env::DISABLE_TUI, default_value_t = true)]
    #[builder(default = true)]
    pub tui: bool,
}

impl TryInto<Vec<String>> for ProcessComposeFlagsUp {
    type Error = String;

    fn try_into(self) -> Result<Vec<String>, Self::Error> {
        let mut args = Vec::new();

        for p in self.config {
            args.push(format!("--{}", arg!(ProcessComposeFlagsUp, config)));
            args.push(p.to_string_lossy().to_string());
        }

        if self.detach_on_success {
            args.push(format!(
                "--{}",
                arg!(ProcessComposeFlagsUp, detach_on_success)
            ));
        }

        #[cfg(unix)]
        if self.detached {
            args.push(format!("--{}", arg!(ProcessComposeFlagsUp, detached)));
        }

        #[cfg(unix)]
        if self.detached_with_tui {
            args.push(format!(
                "--{}",
                arg!(ProcessComposeFlagsUp, detached_with_tui)
            ));
        }

        if self.disable_dotenv {
            args.push(format!("--{}", arg!(ProcessComposeFlagsUp, disable_dotenv)));
        }
        if self.dry_run {
            args.push(format!("--{}", arg!(ProcessComposeFlagsUp, dry_run)));
        }

        for p in self.env_files {
            args.push(format!("--{}", arg!(ProcessComposeFlagsUp, env_files)));
            args.push(p.to_string_lossy().to_string());
        }

        if self.hide_disabled {
            args.push(format!("--{}", arg!(ProcessComposeFlagsUp, hide_disabled)));
        }
        if self.keep_project {
            args.push(format!("--{}", arg!(ProcessComposeFlagsUp, keep_project)));
        }
        if self.logs_truncate {
            args.push(format!("--{}", arg!(ProcessComposeFlagsUp, logs_truncate)));
        }
        for ns in self.namespaces {
            args.push(format!("--{}", arg!(ProcessComposeFlagsUp, namespaces)));
            args.push(ns);
        }
        if self.no_deps {
            args.push(format!("--{}", arg!(ProcessComposeFlagsUp, no_deps)));
        }
        if self.recursive_metrics {
            args.push(format!(
                "--{}",
                arg!(ProcessComposeFlagsUp, recursive_metrics)
            ));
        }

        // refresh rate
        args.push(format!("--{}", arg!(ProcessComposeFlagsUp, refresh_rate)));
        args.push(self.refresh_rate.get().to_string());

        if self.reverse {
            args.push(format!("--{}", arg!(ProcessComposeFlagsUp, reverse)));
        }

        for p in self.shortcuts {
            args.push(format!("--{}", arg!(ProcessComposeFlagsUp, shortcuts)));
            args.push(p.to_string_lossy().to_string());
        }
        if let Some(slow) = self.slow_refresh_rate {
            args.push(format!(
                "--{}",
                arg!(ProcessComposeFlagsUp, slow_refresh_rate)
            ));
            args.push(slow);
        }

        args.push(format!("--{}", arg!(ProcessComposeFlagsUp, sort)));
        args.push(self.sort);

        args.push(format!("--{}", arg!(ProcessComposeFlagsUp, theme)));
        args.push(self.theme);

        if !self.tui {
            args.push(format!(
                "--{}={}",
                arg!(ProcessComposeFlagsUp, tui),
                self.tui
            ));
        }

        Ok(args)
    }
}
