use std::sync::Arc;

use axum::extract;

use crate::http::server::Server;

use crate::http::extractors::AuthTokenExtractor;
use crate::http::response::Response;
use axum::response::Json;
use hyper::StatusCode;
use log::{error, info};
use model::Snowflake;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Debug)]
pub struct RequestBody {
    user: Snowflake,
}

pub async fn vote_tgg_handler(
    auth_token: AuthTokenExtractor,
    body: extract::Json<RequestBody>,
    server: extract::Extension<Arc<Server>>,
) -> (StatusCode, Json<Response>) {
    let (server, body) = (server.0, body.0);

    if auth_token.0 != server.config.tgg_token[..] {
        return (StatusCode::UNAUTHORIZED, generate_invalid_signature());
    }

    if let Err(e) = server.database.add_vote(body.user).await {
        error!("Error while adding vote: {}", e); // TODO: Sentry
        return (
            StatusCode::INTERNAL_SERVER_ERROR,
            Json(Response::error("Database error")),
        );
    }

    info!("Logged vote for {}", body.user);

    (StatusCode::OK, Json(Response::success()))
}

fn generate_invalid_signature() -> Json<Response> {
    generate_unauthorized("Invalid signature")
}

fn generate_unauthorized(error: &str) -> Json<Response> {
    Json(Response::error(error))
}
