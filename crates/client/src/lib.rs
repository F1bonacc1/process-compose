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

/// Use this to get strongly typed HTTP client.
/// Does not support multitype returns(none in Process Compose yet)
/// nor upgrades (101 Switching Protocols).
#[cfg(feature = "progenitor")]
pub fn progenitor_pretty(maybe_config: Option<progenitor::Generator>) -> String {
    let mut generator = maybe_config.unwrap_or_else(|| {
        #[allow(unused_mut)]
        let mut settings = progenitor::GenerationSettings::default();
        #[cfg(feature = "builder")]
        settings.with_interface(progenitor::InterfaceStyle::Builder);
        progenitor::Generator::new(&settings)
    });
    let mut openapi = openapi().clone();
    {
        use openapiv3::{PathItem, ReferenceOr};
        let paths_map = &mut openapi.paths.paths;
        paths_map.retain(|_, item| {
            let ReferenceOr::Item(PathItem {
                get,
                put,
                post,
                delete,
                options,
                head,
                patch,
                trace,
                ..
            }) = item
            else {
                return true;
            };

            let has_101 = [
                get.as_ref(),
                put.as_ref(),
                post.as_ref(),
                delete.as_ref(),
                options.as_ref(),
                head.as_ref(),
                patch.as_ref(),
                trace.as_ref(),
            ]
            .into_iter()
            .flatten()
            .any(|op| {
                op.responses
                    .responses
                    .iter()
                    .any(|(status, _)| matches!(status, openapiv3::StatusCode::Code(101)))
            });
            !has_101
        });
    }

    let tokens = generator.generate_tokens(&openapi).unwrap();
    let ast = syn::parse2(tokens).unwrap();
    prettyplease::unparse(&ast)
}

#[cfg(feature = "ws")]
#[derive(serde::Serialize, serde::Deserialize, Debug)]
pub struct LogMessage {
    pub process_name: String,
    pub message: String,
}

/// Returns URL parts without scheme and host.
/// Output is (path, query_string).
/// Process names are joined with comma, so proces name with comman will add 2 processes.
#[cfg(feature = "ws")]
pub fn process_logs_ws<ProcessName: AsRef<str>>(
    process_names: &[ProcessName],
    offset: usize,
    follow: bool,
) -> (&'static str, String) {
    let names = process_names
        .iter()
        .map(|s| s.as_ref())
        .collect::<Vec<_>>()
        .join(",");
    let mut query = url::form_urlencoded::Serializer::new(String::new());
    query.append_pair("name", &names);
    query.append_pair("offset", &offset.to_string());
    query.append_pair("follow", if follow { "true" } else { "false" });

    ("/process/logs/ws", query.finish())
}
