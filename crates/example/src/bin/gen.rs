// includes generated code
#![allow(renamed_and_removed_lints)]

use crate::openapi::builder::GetProcessInfo;

mod openapi {
    include!(concat!(env!("OUT_DIR"), "/client.rs"));
}

mod config {
    include!(concat!(env!("OUT_DIR"), "/config.rs"));
}

#[tokio::main]
async fn main() {
    // we just compile it to check for compile errors
    let client = crate::openapi::Client::new("locahost:8080");
    if let Ok(response) = GetProcessInfo::new(&client).name("asd").send().await {
        let _name = &response.name;
        unreachable!("errors on bad url");
    }

    let _config = crate::config::Project::builder().processes(crate::config::Processes(<_>::default()));

    println!("Compiles!")
}
