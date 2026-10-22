use resrap_rs::{Resrap, templates};

/// Thin wrapper around resrap-rs' bundled templates. A fresh Resrap instance
/// is used per request to mirror the browser WASM wrapper exactly: same
/// library, same template, same seed and token count produce the same code.
pub struct CodeGen;

impl CodeGen {
    pub fn new() -> Self {
        CodeGen
    }

    pub fn generate(&self, name: &str, seed: u64, tokens: usize) -> String {
        let template = match templates::get(name) {
            Ok(template) => template,
            Err(_) => return String::new(),
        };

        let mut rs = Resrap::new();
        if template.load(&mut rs).is_err() {
            return String::new();
        }

        rs.generate_complete_with_seeded(template.name, "program", seed, tokens)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn builtin_templates_generate() {
        for name in templates::names() {
            let out = CodeGen::new().generate(name, 42, 120);
            assert!(!out.is_empty(), "template {name} produced nothing");
        }
    }

    #[test]
    fn deterministic_and_seed_sensitive() {
        let codegen = CodeGen::new();
        let a = codegen.generate("go", 1234, 120);
        let b = codegen.generate("go", 1234, 120);
        let c = codegen.generate("go", 1235, 120);
        assert_eq!(a, b);
        assert_ne!(a, c);
    }

    #[test]
    fn unknown_template_is_empty() {
        assert!(CodeGen::new().generate("brainfuck", 1, 120).is_empty());
    }
}
