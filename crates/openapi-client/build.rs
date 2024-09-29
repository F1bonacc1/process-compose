fn main() {
    let src = "swagger.3.json";
    println!("cargo:rerun-if-changed={}", src);
    let spec = include_str!("swagger.3.json");
    let spec = serde_json::from_str(spec).unwrap();
    let mut generator = progenitor::Generator::default();
    let tokens = generator.generate_tokens(&spec).unwrap();
    let ast = syn::parse2(tokens).unwrap();
    let content = prettyplease::unparse(&ast);
    let mut out_file = std::path::Path::new(&std::env::var("OUT_DIR").unwrap()).to_path_buf();
    out_file.push("lib.rs");
    std::fs::write(out_file, content).unwrap();
}
