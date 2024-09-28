use process_compose_openapi_client;

#[tokio::test]
async fn compiles_but_errors_with_bad_url() {
    let client = process_compose_openapi_client::Client::new("locahost:8080");
    client.get_host_name().await.expect_err("errors on bad url");
}
