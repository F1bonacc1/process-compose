//! provides generator to use in build.rs, with default `progenitor` provider

use core::cell::OnceCell;
use std::{collections::HashMap, sync::OnceLock};

use openapiv3::OpenAPI;

/// Raw OpenAPI spec string
pub const RAW_JSON: &str = include_str!("../../../src/docs/swagger.json");

static CONFIG: OnceLock<OpenAPI> = OnceLock::new();

/// OpenAPI spec parsed
pub fn openapi() -> &'static OpenAPI {
    CONFIG.get_or_init(|| {
        serde_json::from_str(RAW_JSON).unwrap()
    });
    CONFIG.get().unwrap()
}

//#[cfg(feature = "progenitor")]



pub fn progenitor_pretty(maybe_config: Option<progenitor::Generator>) -> String {
  let mut generator = maybe_config.unwrap_or_default();
  let tokens = generator.generate_tokens(openapi()).unwrap();
  let ast = syn::parse2(tokens).unwrap();
  prettyplease::unparse(&ast)
}
