fn main() {
    let content = process_compose_openapi_client::progenitor_pretty(None);
    let mut out_file = std::path::Path::new(&std::env::var("OUT_DIR").unwrap()).to_path_buf();
    out_file.push("lib.rs");
    std::fs::write(out_file, content).unwrap();
}
