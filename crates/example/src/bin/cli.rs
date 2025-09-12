use process_compose_client::cli::cmd::parent::{ProcessComposeCommand, ProcessComposeFlags};
use process_compose_client::cli::cmd::up::ProcessComposeFlagsUp;
use process_compose_client::cli::exec;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Locate the repository root by walking up until process-compose.yaml is found
    let project_path = {
        let mut dir = std::env::current_dir()?;
        loop {
            if dir.join("process-compose.yaml").is_file() {
                break dir;
            }
            if !dir.pop() {
                return Err("process-compose.yaml not found in any parent directory".into());
            }
        }
    };
    let path = project_path.join("process-compose.yaml");
    let override_path = project_path.join("process-compose.override.yaml");
    let up = ProcessComposeFlagsUp::builder()
        .config(vec![path.clone(), override_path.clone()])
        .tui(true)
        .build();
    let flags = ProcessComposeFlags::builder()
        .subcommand(ProcessComposeCommand::up(up))
        .build();

    let _child = exec::process_compose(flags)
        .unwrap()
        .spawn()?
        .wait()
        .await?;

    Ok(())
}
