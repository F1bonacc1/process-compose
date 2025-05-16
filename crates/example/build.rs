fn main() {
    // openapi client
    let client = process_compose_openapi_client::progenitor_pretty(None);
    let mut out_file = std::path::Path::new(&std::env::var("OUT_DIR").unwrap()).to_path_buf();
    out_file.push("client.rs");
    std::fs::write(out_file, client).unwrap();

    // config schema builder/parser
    let config = process_compose_openapi_client::typify_pretty(None);
    let mut out_file = std::path::Path::new(&std::env::var("OUT_DIR").unwrap()).to_path_buf();
    out_file.push("config.rs");
    std::fs::write(out_file, config).unwrap();
  }
