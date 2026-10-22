use axum::{
    Json, Router,
    extract::State,
    http::StatusCode,
    routing::{get, post},
};
use resrap_rs::templates;
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
    code: String,
}

async fn generate(
    State(state): State<AppState>,
    Json(req): Json<GenerateRequest>,
) -> Result<Json<GenerateResponse>, (StatusCode, String)> {
    if req.tokens == 0 || req.tokens > 400 {
        return Err((
            StatusCode::BAD_REQUEST,
            "tokens must be between 1 and 400".to_string(),
        ));
    }
    let result = state.generator.generate(&req.name, req.seed, req.tokens);
    if result.is_empty() {
        return Err((
            StatusCode::NOT_FOUND,
            format!("unknown template: {}", req.name),
        ));
    }
    Ok(Json(GenerateResponse { code: result }))
}

async fn health() -> Json<serde_json::Value> {
    Json(serde_json::json!({
        "ok": true,
        "languages": templates::names(),
    }))
}

#[tokio::main]
async fn main() {
    let generator = Arc::new(CodeGen::new());
    let state = AppState { generator };

    let app = Router::new()
        .route("/generate", post(generate))
        .route("/health", get(health))
        .with_state(state);

    let bind = std::env::var("CODEGEN_BIND").unwrap_or_else(|_| "0.0.0.0:8081".to_string());
    println!("resrap generator listening on {bind}");
    let listener = tokio::net::TcpListener::bind(&bind).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
