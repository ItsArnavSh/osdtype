use resrap_rs::Resrap;

pub struct CodeGen {
    r: Resrap,
}
impl CodeGen {
    pub fn new() -> Self {
        let mut r = Resrap::new();
        r.parse_grammar_file(String::from("c"), String::from("templates/c.g4"))
            .expect("Could not parse C");
        r.parse_grammar_file(String::from("go"), String::from("templates/go.g4"))
            .expect("Could not parse go");
        r.parse_grammar_file(String::from("cpp"), String::from("templates/cpp.g4"))
            .expect("Could not parse Cpp");
        r.parse_grammar_file(String::from("java"), String::from("templates/java.g4"))
            .expect("Could not parse Java");
        r.parse_grammar_file(String::from("rs"), String::from("templates/rs.g4"))
            .expect("Could not parse Rust");
        r.parse_grammar_file(String::from("ts"), String::from("templates/ts.g4"))
            .expect("Could not parse TS");
        CodeGen { r }
    }
    pub fn generate(&self, name: &str, seed: u64, tokens: usize) -> Vec<String> {
        let res = self
            .r
            .generate_with_seed(name, String::from("program"), seed, tokens);
        match res {
            Ok(str) => return str,
            Err(_) => return vec![],
        }
    }
}
