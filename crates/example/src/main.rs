include!(concat!(env!("OUT_DIR"), "/client.rs"));
include!(concat!(env!("OUT_DIR"), "/config.rs"));

#[tokio::main]
async fn main() {
    let client = crate::Client::new("locahost:8080");
    if let Ok(response) = client.get_process_info("process-compose").await {
        let _name = &response.name;
        unreachable!("errors on bad url");
    }

    let _config = crate::Project::builder().processes(Processes(<_>::default()));

    println!("Compiles!")
}
