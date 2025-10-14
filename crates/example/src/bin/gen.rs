use crate::openapi::builder::GetProcessInfo;

#[allow(mismatched_lifetime_syntaxes)] // seems some rust version detection prevents consistent linting for expect
pub mod openapi {
    include!(concat!(env!("OUT_DIR"), "/client.rs"));
}

#[expect(clippy::derivable_impls)]
#[expect(clippy::clone_on_copy)]
pub mod config {
    include!(concat!(env!("OUT_DIR"), "/config.rs"));
}

#[tokio::main]
async fn main() {
    // we just compile it to check for compile errors
    let client = crate::openapi::Client::new("locahost:8080");
    if let Ok(response) = GetProcessInfo::new(&client).name("process-compose").send().await {
        let _name = &response.name;
        unreachable!("errors on bad url");
    }

    let _config =
        crate::config::Project::builder().processes(crate::config::Processes(<_>::default()));

    println!("Compiles!")
}
