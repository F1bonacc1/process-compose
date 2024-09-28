use progenitor::generate_api;
generate_api!(spec = "../../src/docs/swagger.3.json");

// https://github.com/oxidecomputer/progenitor/issues/694#issuecomment-1901006117

// Compiling process-compose-openapi-client v0.1.0 (/home/dz/github.com/F1bonacc1/process-compose/crates/openapi-client)
// error: generation error for ../../src/docs/swagger.3.json: unexpected or unhandled format in the OpenAPI document path /hostname is missing operation ID
//  --> src/lib.rs:2:22
//   |
// 2 | generate_api!(spec = "../../src/docs/swagger.3.json");
//   |                      ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

// error: could not compile `process-compose-openapi-client` (lib) due to 1 previous error
// warning: build failed, waiting for other jobs to finish...
