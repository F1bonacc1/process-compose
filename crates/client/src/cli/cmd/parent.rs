use clap::{Parser, Subcommand};
use std::path::PathBuf;

use crate::cli::cmd::flags::env;
use crate::cli::cmd::flags::DEFAULT_PORT_NUM;
use crate::cli::cmd::up::ProcessComposeFlagsUp;
/// Process Compose startup flags
#[derive(Debug, Clone, Parser, struct_field_names::StructFieldNames, bon::Builder)]
pub struct ProcessComposeFlags {
    /// Specify the log file path (env: PC_LOG_FILE)
    /// Default "/tmp/process-compose-<user>.log"
    #[arg(long = "log-file", env = "PC_LOG_FILE")]
    pub log_file: Option<PathBuf>,

    /// Disable HTTP server (env: PC_NO_SERVER)
    #[arg(long = "no-server", env = env::NO_SERVER)]
    #[builder(default = false)]
    pub no_server: bool,

    /// Shut down processes in reverse dependency order (env: PC_ORDERED_SHUTDOWN)
    #[arg(long = "ordered-shutdown", env = env::ORDERED_SHUTDOWN)]
    #[builder(default = false)]
    pub ordered_shutdown: bool,

    /// Port number (env: PC_PORT_NUM)
    #[arg(long = "port", env = env::PORT_NUM, default_value_t = DEFAULT_PORT_NUM)]
    #[builder(default = DEFAULT_PORT_NUM)]
    pub port: u16,

    /// Enable read-only mode (env: PC_READ_ONLY)
    #[arg(long = "read-only", env = env::READ_ONLY)]
    #[builder(default = false)]
    pub read_only: bool,

    /// Path to unix socket (env: PC_SOCKET_PATH)
    /// Default "/tmp/process-compose-<pid>.sock"
    #[arg(long = "unix-socket", env = env::SOCKET_PATH)]
    pub unix_socket: Option<PathBuf>,

    /// Use unix domain sockets instead of TCP
    #[arg(long = "use-uds", default_value_t = false)]
    #[builder(default = false)]
    pub use_uds: bool,

    #[command(subcommand)]
    pub subcommand: Option<ProcessComposeCommand>,
}

#[derive(Debug, Clone, Subcommand, struct_field_names::EnumVariantNames)]
pub enum ProcessComposeCommand {
    /// Run process compose project
    Up(ProcessComposeFlagsUp),
}

impl ProcessComposeCommand {
    pub fn up(value: ProcessComposeFlagsUp) -> Self {
        Self::Up(value)
    }
}

impl TryInto<Vec<String>> for ProcessComposeCommand {
    type Error = String;

    fn try_into(self) -> Result<Vec<String>, Self::Error> {
        let mut args = Vec::new();
        match self {
            ProcessComposeCommand::Up(up) => {
                args.push("up".to_string());
                let up_args: Vec<String> = up.try_into()?;
                args.extend(up_args);
            }
        }
        Ok(args)
    }
}

impl TryInto<Vec<String>> for ProcessComposeFlags {
    type Error = String;

    fn try_into(self) -> Result<Vec<String>, Self::Error> {
        let mut args = Vec::new();

        if let Some(path) = self.log_file {
            args.push(format!("--{}", arg!(ProcessComposeFlags, log_file)));
            args.push(path.to_string_lossy().to_string());
        }
        if self.no_server {
            args.push(format!("--{}", arg!(ProcessComposeFlags, no_server)));
        }
        if self.ordered_shutdown {
            args.push(format!("--{}", arg!(ProcessComposeFlags, ordered_shutdown)));
        }

        // Always include port to be explicit
        args.push(format!("--{}", arg!(ProcessComposeFlags, port)));
        args.push(self.port.to_string());

        if self.read_only {
            args.push(format!("--{}", arg!(ProcessComposeFlags, read_only)));
        }
        if let Some(sock) = self.unix_socket {
            args.push(format!("--{}", arg!(ProcessComposeFlags, unix_socket)));
            args.push(sock.to_string_lossy().to_string());
        }
        if self.use_uds {
            args.push(format!("--{}", arg!(ProcessComposeFlags, use_uds)));
        }

        if let Some(sub) = self.subcommand {
            let mut sub_args: Vec<String> = sub.try_into()?;
            args.append(&mut sub_args);
        }

        Ok(args)
    }
}
