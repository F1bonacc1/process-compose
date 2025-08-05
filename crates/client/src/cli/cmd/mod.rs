// Define macro before submodules so it's in scope for them
macro_rules! arg {
    ($ty:ty, $field:ident) => {{
        <$ty as clap::CommandFactory>::command()
            .get_arguments()
            .find(|a| a.get_id() == <$ty>::FIELD_NAMES.$field)
            .and_then(|a| a.get_long())
            .expect("long argument name not found")
    }};
}

pub mod flags;
pub mod parent;
pub mod up;
