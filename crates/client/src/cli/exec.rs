//! Execute process compose process.
use process_wrap::tokio::*;

/// Environment variable for the path to the process compose binary,
/// if defined, used instead of default OS and process search strategy.
pub const ENV_PC_BIN: &str = "PC_BIN";

/// Comand line fully controlled by
/// - OS binary path search or `PC_BIN`
/// - command line arguments generated from `ProcessComposeFlags`
///
/// Returns a `CommandWrap` that can be used to interact with the spawned process, including adding wrappers.
pub fn process_compose(
    command: super::cmd::parent::ProcessComposeFlags,
) -> Result<CommandWrap, String> {
    let args: Vec<String> = command.try_into()?;
    let mut command = "process-compose".to_string();
    if let Some(pc_bin) = std::env::var_os(ENV_PC_BIN) {
        command = pc_bin.to_string_lossy().to_string();
    }

    let command = CommandWrap::with_new(command, |command| {
        command.args(&args);
    });

    Ok(command)
}
