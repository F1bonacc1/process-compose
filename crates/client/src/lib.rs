//! provides generator to use in build.rs, with default `progenitor` provider

use openapiv3::OpenAPI;
use std::sync::OnceLock;

#[cfg(feature = "cli")]
pub mod cli;

/// Raw OpenAPI spec string
pub const OPENAPI_JSON_STRING: &str = include_str!("../../../src/docs/swagger.json");

/// Raw Process Compose config file schema string
pub const CONFIG_SCHEMA_JSON_STRING: &str =
    include_str!("../../../schemas/process-compose-schema.json");

static OPENAPI: OnceLock<OpenAPI> = OnceLock::new();

static CONFIG_SCHEMA: OnceLock<schemars::schema::RootSchema> = OnceLock::new();

/// OpenAPI spec parsed
pub fn openapi() -> &'static OpenAPI {
    OPENAPI.get_or_init(|| serde_json::from_str(OPENAPI_JSON_STRING).unwrap());
    OPENAPI.get().unwrap()
}

/// Parsed Process Compose config file schema
pub fn config_schema() -> &'static schemars::schema::RootSchema {
    CONFIG_SCHEMA.get_or_init(|| serde_json::from_str(CONFIG_SCHEMA_JSON_STRING).unwrap());
    CONFIG_SCHEMA.get().unwrap()
}

/// Use this to get storngly typed config file builder and parser
#[cfg(feature = "typify")]
pub fn typify_pretty(maybe_config: Option<typify::TypeSpace>) -> String {
    use typify::{TypeSpace, TypeSpaceSettings};
    let mut type_space = maybe_config
        .unwrap_or_else(|| TypeSpace::new(TypeSpaceSettings::default().with_struct_builder(true)));
    type_space.add_root_schema(config_schema().clone()).unwrap();
    prettyplease::unparse(&syn::parse2::<syn::File>(type_space.to_stream()).unwrap())
}

/// Use this to get strongly typed HTTP client
#[cfg(feature = "progenitor")]
pub fn progenitor_pretty(maybe_config: Option<progenitor::Generator>) -> String {
    let mut generator = maybe_config.unwrap_or_default();
    let tokens = generator.generate_tokens(openapi()).unwrap();
    let ast = syn::parse2(tokens).unwrap();
    prettyplease::unparse(&ast)
}
