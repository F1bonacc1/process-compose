use process_compose_openapi_client;

#[tokio::test]
async fn compiles_but_errors_with_bad_url() {
    let client = process_compose_openapi_client::Client::new("locahost:8080");
    match client.get_process_info("process-compose").await {
        Ok(response) => {
            let _name = &response.name;
            unreachable!("errors on bad url");
        }
        Err(_) => {}
    }
}
