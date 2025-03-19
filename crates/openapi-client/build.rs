fn main() {
    let spec = if option_env!("CARGO_MANIFEST_DIR").is_some()
        && option_env!("CARGO_REGISTRY_TOKEN").is_some()
    {
        println!("cargo:rerun-if-changed=../../../src/docs/swagger.json");
        std::fs::read_to_string("../../../src/docs/swagger.json").unwrap()
    } else {
        println!("cargo:rerun-if-changed=../../src/docs/swagger.json");
        std::fs::read_to_string("../../src/docs/swagger.json").unwrap()
    };
    let spec = serde_json::from_str(&spec).unwrap();
    let mut generator = progenitor::Generator::default();
    let tokens = generator.generate_tokens(&spec).unwrap();
    let ast = syn::parse2(tokens).unwrap();
    let content = prettyplease::unparse(&ast);
    let mut out_file = std::path::Path::new(&std::env::var("OUT_DIR").unwrap()).to_path_buf();
    out_file.push("lib.rs");
    std::fs::write(out_file, content).unwrap();
}
