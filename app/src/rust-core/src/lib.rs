use resrap_rs::{Resrap, templates};
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub fn add(a: i32, b: i32) -> i32 {
    a + b
}

/// Short names of every language bundled with resrap-rs.
#[wasm_bindgen]
pub fn languages() -> Vec<String> {
    templates::names()
        .into_iter()
        .map(String::from)
        .collect()
}

/// Generates a complete snippet for `lang` (a bundled template name such as
/// `"c"`, `"cpp"`, `"python"` or `"js"`), deterministically from `seed`.
#[wasm_bindgen]
pub fn generate(lang: String, seed: u32, tokens: u32) -> String {
    let template = match templates::get(lang.as_str()) {
        Ok(template) => template,
        Err(_) => return String::new(),
    };

    let mut rs = Resrap::new();
    if template.load(&mut rs).is_err() {
        return String::new();
    }

    rs.generate_complete_with_seeded(template.name, "program", seed as u64, tokens as usize)
}
