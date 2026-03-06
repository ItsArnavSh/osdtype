use resrap_rs::Resrap;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub fn add(a: i32, b: i32) -> i32 {
    a + b
}

#[wasm_bindgen]
pub fn generate(grammar: String, seed: u32, tokens: u32) -> Vec<String> {
    let mut rs = Resrap::new();
    match rs.parse_grammar(String::from("a"), grammar) {
        Ok(_) => {
            match rs.generate_with_seed("a", String::from("program"), seed as u64, tokens as usize)
            {
                Ok(list) => return list,
                Err(_) => return vec![],
            }
        }
        Err(_) => return vec![],
    }
}
