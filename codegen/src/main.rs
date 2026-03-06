use axum::{Json, Router, extract::State, routing::post};
use serde::{Deserialize, Serialize};
use std::sync::Arc;

mod resrap;
use crate::resrap::CodeGen;

#[derive(Clone)]
struct AppState {
    generator: Arc<CodeGen>,
}

#[derive(Deserialize)]
struct GenerateRequest {
    name: String,
    seed: u64,
    tokens: usize,
}

#[derive(Serialize)]
struct GenerateResponse {
    code: Vec<String>,
}

async fn generate(
    State(state): State<AppState>,
    Json(req): Json<GenerateRequest>,
) -> Json<GenerateResponse> {
    let result = state.generator.generate(&req.name, req.seed, req.tokens);

    Json(GenerateResponse { code: result })
}

#[tokio::main]
async fn main() {
    let generator = Arc::new(CodeGen::new());

    let state = AppState { generator };

    let app = Router::new()
        .route("/generate", post(generate))
        .with_state(state);

    println!("Rust generator running on 127.0.0.1:8081");

    let listener = tokio::net::TcpListener::bind("127.0.0.1:8081")
        .await
        .unwrap();

    axum::serve(listener, app).await.unwrap();
}
